package serviceaichat

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/cockroachdb/errors"
	"github.com/forbearing/gst/database"
	modelaichat "github.com/forbearing/gst/internal/model/aichat"
	"github.com/forbearing/gst/model"
	"github.com/forbearing/gst/service"
	"github.com/forbearing/gst/types"
)

type ProviderTestConn struct {
	service.Base[*model.Empty, *modelaichat.Provider, *modelaichat.ProviderTestConnRsp]
}

// Create tests the connection to the AI provider
func (s *ProviderTestConn) Create(ctx *types.ServiceContext, req *modelaichat.Provider) (*modelaichat.ProviderTestConnRsp, error) {
	log := s.WithServiceContext(ctx, ctx.GetPhase())

	if len(req.ID) == 0 {
		return nil, errors.New("provider id is required")
	}

	aiProvider := new(modelaichat.Provider)
	if err := database.Database[*modelaichat.Provider](ctx.DatabaseContext()).
		WithQuery(&modelaichat.Provider{Base: model.Base{ID: req.ID}}).
		First(aiProvider); err != nil {
		return nil, errors.Wrapf(err, "failed to get provider: %s", req.ID)
	}

	log.Infow("testing provider connection", "provider_id", aiProvider.ID, "provider_type", aiProvider.Type)

	// Create HTTP client with timeout
	httpClient := &http.Client{
		Timeout: 10 * time.Second,
	}

	// Get provider configuration
	config := aiProvider.Config.Data()

	// Test connection based on provider type
	var success bool
	var message string

	switch aiProvider.Type {
	case modelaichat.ProviderOpenAI:
		// Test OpenAI-compatible API
		baseURL := config.BaseURL
		if baseURL == "" {
			baseURL = "https://api.openai.com/v1"
		}
		testURL := fmt.Sprintf("%s/models", baseURL)
		req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, testURL, nil)
		if err != nil {
			return nil, errors.Wrap(err, "failed to create request")
		}
		if config.APIKey != "" {
			req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", config.APIKey))
		}
		if config.OrgID != "" {
			req.Header.Set("OpenAI-Organization", config.OrgID)
		}
		// Add extra headers
		for k, v := range config.ExtraHeaders {
			req.Header.Set(k, v)
		}

		resp, err := httpClient.Do(req)
		if err != nil {
			success = false
			message = fmt.Sprintf("connection failed: %v", err)
		} else {
			resp.Body.Close()
			if resp.StatusCode == http.StatusOK {
				success = true
				message = "connection successful"
			} else {
				success = false
				message = fmt.Sprintf("connection failed with status: %d", resp.StatusCode)
			}
		}

	case modelaichat.ProviderAnthropic:
		// Test Anthropic API
		baseURL := config.BaseURL
		if baseURL == "" {
			baseURL = "https://api.anthropic.com/v1"
		}
		testURL := fmt.Sprintf("%s/messages", baseURL)
		req, err := http.NewRequestWithContext(context.Background(), http.MethodPost, testURL, nil)
		if err != nil {
			return nil, errors.Wrap(err, "failed to create request")
		}
		if config.APIKey != "" {
			req.Header.Set("x-api-key", config.APIKey)
			req.Header.Set("anthropic-version", "2023-06-01")
		}
		for k, v := range config.ExtraHeaders {
			req.Header.Set(k, v)
		}

		resp, err := httpClient.Do(req)
		if err != nil {
			success = false
			message = fmt.Sprintf("connection failed: %v", err)
		} else {
			resp.Body.Close()
			if resp.StatusCode == http.StatusOK || resp.StatusCode == http.StatusBadRequest {
				// BadRequest might mean auth is working but request is invalid
				success = true
				message = "connection successful"
			} else {
				success = false
				message = fmt.Sprintf("connection failed with status: %d", resp.StatusCode)
			}
		}

	case modelaichat.ProviderGoogle:
		// Test Google Gemini API
		baseURL := config.BaseURL
		if baseURL == "" {
			baseURL = "https://generativelanguage.googleapis.com/v1beta"
		}
		testURL := fmt.Sprintf("%s/models", baseURL)
		req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, testURL, nil)
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
			success = false
			message = fmt.Sprintf("connection failed: %v", err)
		} else {
			resp.Body.Close()
			if resp.StatusCode == http.StatusOK {
				success = true
				message = "connection successful"
			} else {
				success = false
				message = fmt.Sprintf("connection failed with status: %d", resp.StatusCode)
			}
		}

	case modelaichat.ProviderLocal:
		// Test local provider (Ollama, etc.)
		baseURL := config.BaseURL
		if baseURL == "" {
			baseURL = "http://localhost:11434"
		}
		testURL := fmt.Sprintf("%s/api/tags", baseURL)
		req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, testURL, nil)
		if err != nil {
			return nil, errors.Wrap(err, "failed to create request")
		}
		for k, v := range config.ExtraHeaders {
			req.Header.Set(k, v)
		}
		resp, err := httpClient.Do(req)
		if err != nil {
			success = false
			message = fmt.Sprintf("connection failed: %v", err)
		} else {
			resp.Body.Close()
			if resp.StatusCode == http.StatusOK {
				success = true
				message = "connection successful"
			} else {
				success = false
				message = fmt.Sprintf("connection failed with status: %d", resp.StatusCode)
			}
		}

	default:
		// For other provider types, just check if base URL is accessible
		baseURL := config.BaseURL
		if baseURL == "" {
			success = false
			message = "base URL is not configured"
		} else {
			req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, baseURL, nil)
			if err != nil {
				success = false
				message = fmt.Sprintf("invalid base URL: %v", err)
			} else {
				for k, v := range config.ExtraHeaders {
					req.Header.Set(k, v)
				}
				resp, err := httpClient.Do(req)
				if err != nil {
					success = false
					message = fmt.Sprintf("connection failed: %v", err)
				} else {
					resp.Body.Close()
					success = resp.StatusCode < 500
					if success {
						message = "connection successful"
					} else {
						message = fmt.Sprintf("connection failed with status: %d", resp.StatusCode)
					}
				}
			}
		}
	}

	log.Infow("provider connection test completed", "success", success, "message", message)

	return &modelaichat.ProviderTestConnRsp{
		Success: success,
		Message: message,
	}, nil
}
