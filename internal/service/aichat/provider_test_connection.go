package serviceaichat

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/cockroachdb/errors"
	modelaichat "github.com/forbearing/gst/internal/model/aichat"
	"github.com/forbearing/gst/model"
	"github.com/forbearing/gst/service"
	"github.com/forbearing/gst/types"
)

// TestConnection tests provider connectivity without persisting anything.
type TestConnection struct {
	service.Base[*model.Empty, *modelaichat.Provider, *modelaichat.ProviderTestRsp]
}

// Create tests the connection to the AI provider.
func (s *TestConnection) Create(ctx *types.ServiceContext, provider *modelaichat.Provider) (*modelaichat.ProviderTestRsp, error) {
	log := s.WithServiceContext(ctx, ctx.GetPhase())
	log.Infow("testing provider connection", "provider_id", provider.ID, "provider_type", provider.Type)

	httpClient := &http.Client{
		Timeout: 10 * time.Second,
	}

	config := provider.Config.Data()
	var success bool
	var message string

	switch provider.Type {
	case modelaichat.ProviderOpenAI, modelaichat.ProviderCustom:
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

	case modelaichat.ProviderLocal:
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

	return &modelaichat.ProviderTestRsp{
		Success: success,
		Message: message,
	}, nil
}
