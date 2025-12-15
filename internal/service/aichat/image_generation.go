package serviceaichat

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/cockroachdb/errors"
	"github.com/forbearing/gst/config"
	"github.com/forbearing/gst/database"
	modelaichat "github.com/forbearing/gst/internal/model/aichat"
	"github.com/forbearing/gst/model"
	"github.com/forbearing/gst/provider/minio"
	"github.com/forbearing/gst/service"
	"github.com/forbearing/gst/types"
	"github.com/forbearing/gst/util"
)

type ImageGeneration struct {
	service.Base[*model.Empty, *modelaichat.ImageGenerationReq, *modelaichat.ImageGenerationRsp]
}

func (s *ImageGeneration) Create(ctx *types.ServiceContext, req *modelaichat.ImageGenerationReq) (*modelaichat.ImageGenerationRsp, error) {
	log := s.WithServiceContext(ctx, ctx.GetPhase())
	log.Infow("image generation", "prompt", req.Prompt, "model", req.Model)

	// 1. Get model information
	var aiModel *modelaichat.Model
	var err error

	if len(req.Model) > 0 {
		aiModel = new(modelaichat.Model)
		// Try to find by ModelID first (e.g. "dall-e-3")
		// The database stores "model_id" which matches the provider's model ID
		if err = database.Database[*modelaichat.Model](ctx.DatabaseContext()).
			WithQuery(&modelaichat.Model{ModelID: req.Model}).
			First(aiModel); err != nil {
			return nil, errors.Wrapf(err, "failed to get ai model: %s", req.Model)
		}
	} else {
		// Find default image model
		aiModel = new(modelaichat.Model)
		if err = database.Database[*modelaichat.Model](ctx.DatabaseContext()).
			WithQuery(&modelaichat.Model{Type: modelaichat.ModelTypeImage, IsDefault: util.ValueOf(true)}).
			First(aiModel); err != nil {
			// If no default, try any image model
			if err = database.Database[*modelaichat.Model](ctx.DatabaseContext()).
				WithQuery(&modelaichat.Model{Type: modelaichat.ModelTypeImage}).
				First(aiModel); err != nil {
				return nil, errors.Wrap(err, "no image generation model found")
			}
		}
		// Set the model in request if it was empty
		req.Model = aiModel.ModelID
	}

	// 2. Get provider information
	provider := new(modelaichat.Provider)
	if err = database.Database[*modelaichat.Provider](ctx.DatabaseContext()).Get(provider, aiModel.ProviderID); err != nil {
		return nil, errors.Wrapf(err, "failed to get provider: %s", aiModel.ProviderID)
	}

	// 3. Check provider type
	// Currently we only implement OpenAI-compatible image generation
	if provider.Type != modelaichat.ProviderOpenAI && provider.Type != modelaichat.ProviderCustom {
		return nil, errors.Newf("unsupported provider type for image generation: %s", provider.Type)
	}

	// 4. Prepare request to Provider
	providerConfig := provider.Config.Data()
	apiKey := providerConfig.APIKey
	baseURL := providerConfig.BaseURL
	if baseURL == "" {
		baseURL = "https://api.openai.com/v1"
	}
	// Ensure BaseURL doesn't end with slash
	baseURL = strings.TrimRight(baseURL, "/")

	url := baseURL + "/images/generations"

	// Prepare request body
	// We can mostly use the request object as is, but we might need to adjust some defaults
	if req.N == 0 {
		req.N = 1
	}
	if req.Size == "" {
		req.Size = "1024x1024"
	}

	reqBytes, err := json.Marshal(req)
	if err != nil {
		return nil, errors.Wrap(err, "failed to marshal request")
	}

	client := &http.Client{
		Timeout: 60 * time.Second,
	}

	httpReq, err := http.NewRequestWithContext(ctx.Context(), "POST", url, bytes.NewBuffer(reqBytes))
	if err != nil {
		return nil, errors.Wrap(err, "failed to create http request")
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+apiKey)

	// Add extra headers if any
	for k, v := range providerConfig.ExtraHeaders {
		httpReq.Header.Set(k, v)
	}

	// 5. Send request
	resp, err := client.Do(httpReq)
	if err != nil {
		return nil, errors.Wrap(err, "failed to send request to provider")
	}
	defer resp.Body.Close()

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, errors.Wrap(err, "failed to read response body")
	}

	if resp.StatusCode != http.StatusOK {
		return nil, errors.Newf("provider returned error: status=%d, body=%s", resp.StatusCode, string(bodyBytes))
	}

	// 6. Parse response
	var openAIResp modelaichat.ImageGenerationRsp
	if err := json.Unmarshal(bodyBytes, &openAIResp); err != nil {
		return nil, errors.Wrap(err, "failed to unmarshal response")
	}

	// 7. Save to database (optional, but good for history)
	// We save each generated image as a record
	for i := range openAIResp.Data {
		data := &openAIResp.Data[i]

		// Try to upload to MinIO if enabled
		if config.App.Minio.Enable {
			var imageData []byte
			imageExt := ".png" // Default to png

			if data.B64JSON != "" {
				// Decode base64
				decoded, err := base64.StdEncoding.DecodeString(data.B64JSON)
				if err != nil {
					log.Errorw("failed to decode base64 image", "error", err)
				} else {
					imageData = decoded
				}
			} else if data.URL != "" {
				// Download from URL
				resp, err := http.Get(string(data.URL))
				if err != nil {
					log.Errorw("failed to download image from openai", "url", data.URL, "error", err)
				} else {
					defer resp.Body.Close()
					if resp.StatusCode == http.StatusOK {
						imageData, err = io.ReadAll(resp.Body)
						if err != nil {
							log.Errorw("failed to read image body", "error", err)
						}
					} else {
						log.Errorw("failed to download image", "status", resp.StatusCode)
					}
				}
			}

			if len(imageData) > 0 {
				objectKey := fmt.Sprintf("ai-generated/%s/%s%s", time.Now().Format("2006/01/02"), util.UUID(), imageExt)
				_, err := minio.Put(ctx.Context(), objectKey, bytes.NewReader(imageData))
				if err != nil {
					log.Errorw("failed to upload image to minio", "key", objectKey, "error", err)
				} else {
					// Generate presigned URL (valid for 7 days)
					minioURL, err := minio.PresignedGetURL(ctx.Context(), objectKey, 7*24*time.Hour)
					if err != nil {
						log.Errorw("failed to generate minio presigned url", "key", objectKey, "error", err)
					} else {
						data.URL = minioURL
					}
				}
			}
		}

		imageGen := &modelaichat.ImageGeneration{
			UserID:        ctx.UserID,
			Prompt:        req.Prompt,
			Model:         req.Model,
			Size:          req.Size,
			Quality:       req.Quality,
			Style:         req.Style,
			ImageURL:      string(data.URL),
			RevisedPrompt: data.RevisedPrompt,
		}

		if err := database.Database[*modelaichat.ImageGeneration](ctx.DatabaseContext()).Create(imageGen); err != nil {
			log.Errorw("failed to save image generation record", "error", err)
			// Don't fail the request just because saving history failed
		}
	}

	return &openAIResp, nil
}
