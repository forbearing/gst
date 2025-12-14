package serviceaichat

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/cockroachdb/errors"
	"github.com/forbearing/gst/database"
	"github.com/forbearing/gst/internal/dao"
	modelaichat "github.com/forbearing/gst/internal/model/aichat"
	"github.com/forbearing/gst/model"
	"github.com/forbearing/gst/service"
	"github.com/forbearing/gst/types"
)

// ProviderSyncModels fetches provider models and persists them into database.
type ProviderSyncModels struct {
	service.Base[*model.Empty, *modelaichat.Provider, *modelaichat.ProviderSyncModelsRsp]
}

// Create syncs provider models into the database.
func (s *ProviderSyncModels) Create(ctx *types.ServiceContext, req *modelaichat.Provider) (*modelaichat.ProviderSyncModelsRsp, error) {
	log := s.WithServiceContext(ctx, ctx.GetPhase())

	if len(req.ID) == 0 {
		return nil, errors.New("provider id is required")
	}

	dbProvider := new(modelaichat.Provider)
	if err := database.Database[*modelaichat.Provider](ctx.DatabaseContext()).Get(dbProvider, req.ID); err != nil {
		return nil, errors.Wrapf(err, "failed to get provider: %s", req.ID)
	}

	models, err := fetchProviderModels(ctx, dbProvider)
	if err != nil {
		return nil, errors.Wrap(err, "failed to fetch provider models")
	}

	var created, updated, failed int

	modelIDMap, err := dao.QueryModelsToMap(ctx.DatabaseContext(),
		func(m *modelaichat.Model) string { return m.ModelID },
		func() *modelaichat.Model { return &modelaichat.Model{ProviderID: req.ID} })
	if err != nil {
		return nil, errors.Wrap(err, "failed to query models")
	}

	for _, m := range models {
		if _, exists := modelIDMap[m.ModelID]; exists {
			updated++
		} else {
			created++
		}
		m.ProviderID = req.ID
		if err := database.Database[*modelaichat.Model](ctx.DatabaseContext()).Update(m); err != nil {
			log.Errorw("failed to update model", "model_id", m.ModelID, "error", err)
			failed++
		}
	}

	log.Infow("provider models synced", "total", len(models), "created", created, "updated", updated, "failed", failed)

	return &modelaichat.ProviderSyncModelsRsp{
		Total:   len(models),
		Created: created,
		Updated: updated,
		Failed:  failed,
	}, nil
}

// fetchProviderModels retrieves provider models from remote API.
func fetchProviderModels(ctx *types.ServiceContext, provider *modelaichat.Provider) ([]*modelaichat.Model, error) {
	httpClient := &http.Client{
		Timeout: 30 * time.Second,
	}

	config := provider.Config.Data()
	var models []*modelaichat.Model

	switch provider.Type {
	case modelaichat.ProviderOpenAI:
		/*
			export OPENAI_API_KEY="sk-..."
			export OPENAI_ORG_ID="org-..."  # optional

			curl https://api.openai.com/v1/models \
			  -H "Authorization: Bearer $OPENAI_API_KEY" \
			  -H "OpenAI-Organization: $OPENAI_ORG_ID"
		*/
		baseURL := config.BaseURL
		if baseURL == "" {
			baseURL = "https://api.openai.com/v1"
		}
		modelsURL := fmt.Sprintf("%s/models", baseURL)
		req, err := http.NewRequestWithContext(ctx.Context(), http.MethodGet, modelsURL, nil)
		if err != nil {
			return nil, errors.Wrap(err, "failed to create request")
		}
		if config.APIKey != "" {
			req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", config.APIKey))
		}
		if config.OrgID != "" {
			req.Header.Set("OpenAI-Organization", config.OrgID)
		}
		for k, v := range config.ExtraHeaders {
			req.Header.Set(k, v)
		}

		resp, err := httpClient.Do(req)
		if err != nil {
			return nil, errors.Wrap(err, "failed to fetch models")
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			return nil, errors.Newf("failed to fetch models: status %d", resp.StatusCode)
		}

		var openaiResp struct {
			Data []struct {
				ID      string `json:"id"`
				Object  string `json:"object"`
				Created int64  `json:"created"`
				OwnedBy string `json:"owned_by"`
			} `json:"data"`
		}
		if err := json.NewDecoder(resp.Body).Decode(&openaiResp); err != nil {
			return nil, errors.Wrap(err, "failed to decode response")
		}
		for _, m := range openaiResp.Data {
			models = append(models, &modelaichat.Model{
				Base:        model.Base{ID: m.ID},
				Name:        m.ID,
				ModelID:     m.ID,
				Type:        modelaichat.ModelTypeChat,
				Description: m.ID,
			})
		}

	case modelaichat.ProviderAnthropic:
		/*
			export ANTHROPIC_API_KEY="sk-..."
			curl https://api.anthropic.com/v1/models \
				-H "x-api-key: $ANTHROPIC_API_KEY" \
				-H "anthropic-version: 2023-06-01"
		*/
		baseURL := config.BaseURL
		if baseURL == "" {
			baseURL = "https://api.anthropic.com/v1"
		}
		modelsURL := fmt.Sprintf("%s/models", baseURL)
		req, err := http.NewRequestWithContext(ctx.Context(), http.MethodGet, modelsURL, nil)
		if err != nil {
			return nil, errors.Wrap(err, "failed to create request")
		}
		if config.APIKey != "" {
			req.Header.Set("x-api-key", config.APIKey)
		}
		req.Header.Set("anthropic-version", "2023-06-01")
		for k, v := range config.ExtraHeaders {
			req.Header.Set(k, v)
		}

		resp, err := httpClient.Do(req)
		if err != nil {
			return nil, errors.Wrap(err, "failed to fetch models")
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			return nil, errors.Newf("failed to fetch models: status %d", resp.StatusCode)
		}

		var anthropicResp struct {
			Data []struct {
				ID          string `json:"id"`
				DisplayName string `json:"display_name"`
				CreatedAt   string `json:"created_at"`
				Type        string `json:"type"`
			} `json:"data"`
		}
		if err := json.NewDecoder(resp.Body).Decode(&anthropicResp); err != nil {
			return nil, errors.Wrap(err, "failed to decode response")
		}
		for _, m := range anthropicResp.Data {
			models = append(models, &modelaichat.Model{
				Base:        model.Base{ID: m.ID},
				ModelID:     m.ID,
				Name:        m.DisplayName,
				Type:        modelaichat.ModelTypeChat,
				Description: m.DisplayName,
			})
		}

	case modelaichat.ProviderGoogle:
		/*
			export GEMINI_API_KEY="AIza..."
			curl "https://generativelanguage.googleapis.com/v1beta/models?key=$GEMINI_API_KEY"
		*/
		baseURL := config.BaseURL
		if baseURL == "" {
			baseURL = "https://generativelanguage.googleapis.com/v1beta"
		}
		modelsURL := fmt.Sprintf("%s/models", baseURL)
		req, err := http.NewRequestWithContext(ctx.Context(), http.MethodGet, modelsURL, nil)
		if err != nil {
			return nil, errors.Wrap(err, "failed to create request")
		}
		if config.APIKey != "" {
			q := req.URL.Query()
			q.Set("key", config.APIKey)
			req.URL.RawQuery = q.Encode()
		}
		for k, v := range config.ExtraHeaders {
			req.Header.Set(k, v)
		}

		resp, err := httpClient.Do(req)
		if err != nil {
			return nil, errors.Wrap(err, "failed to fetch models")
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			return nil, errors.Newf("failed to fetch models: status %d", resp.StatusCode)
		}

		var geminiResp struct {
			Models []struct {
				Name                       string   `json:"name"`
				DisplayName                string   `json:"displayName"`
				Description                string   `json:"description"`
				Version                    string   `json:"version"`
				SupportedGenerationMethods []string `json:"supportedGenerationMethods"`
			} `json:"models"`
		}
		if err := json.NewDecoder(resp.Body).Decode(&geminiResp); err != nil {
			return nil, errors.Wrap(err, "failed to decode response")
		}

		for _, m := range geminiResp.Models {
			// // Only include models that support generateContent
			// if !slices.Contains(m.SupportedGenerationMethods, "generateContent") {
			// 	continue
			// }

			// sync all models

			// remove the prefix "models/"
			// the model name like: "models/gemini-2.5-flash-image"
			modelName := strings.TrimPrefix(m.Name, "models/")
			if modelName == "" {
				modelName = m.DisplayName
			}
			models = append(models, &modelaichat.Model{
				Base:        model.Base{ID: modelName},
				ModelID:     modelName,
				Name:        m.DisplayName,
				Type:        modelaichat.ModelTypeChat,
				Description: m.Description,
			})
		}

	case modelaichat.ProviderLocal:
		/*
			export OLLAMA_BASE_URL="http://localhost:11434"
			curl $OLLAMA_BASE_URL/api/tags
		*/
		baseURL := config.BaseURL
		if baseURL == "" {
			baseURL = "http://localhost:11434"
		}
		modelsURL := fmt.Sprintf("%s/api/tags", baseURL)
		req, err := http.NewRequestWithContext(ctx.Context(), http.MethodGet, modelsURL, nil)
		if err != nil {
			return nil, errors.Wrap(err, "failed to create request")
		}
		for k, v := range config.ExtraHeaders {
			req.Header.Set(k, v)
		}

		resp, err := httpClient.Do(req)
		if err != nil {
			return nil, errors.Wrap(err, "failed to fetch models")
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			return nil, errors.Newf("failed to fetch models: status %d", resp.StatusCode)
		}

		var ollamaResp struct {
			Models []struct {
				Name       string `json:"name"`
				Model      string `json:"model"`
				ModifiedAt string `json:"modified_at"`
				Size       int64  `json:"size"`
				Digest     string `json:"digest"`
				Details    struct {
					Format            string   `json:"format"`
					Family            string   `json:"family"`
					Families          []string `json:"families"`
					ParameterSize     string   `json:"parameter_size"`
					QuantizationLevel string   `json:"quantization_level"`
					ContextSize       int      `json:"context_size"`
				} `json:"details"`
			} `json:"models"`
		}
		if err := json.NewDecoder(resp.Body).Decode(&ollamaResp); err != nil {
			return nil, errors.Wrap(err, "failed to decode response")
		}
		for _, m := range ollamaResp.Models {
			models = append(models, &modelaichat.Model{
				Base:        model.Base{ID: m.Model},
				ModelID:     m.Model,
				Name:        m.Name,
				Type:        modelaichat.ModelTypeChat,
				Description: m.Name,
			})
		}

	default:
		return nil, errors.Newf("unsupported provider type: %s", provider.Type)
	}

	return models, nil
}
