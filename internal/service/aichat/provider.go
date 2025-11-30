package serviceaichat

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/cockroachdb/errors"
	modelaichat "github.com/forbearing/gst/internal/model/aichat"
	"github.com/forbearing/gst/model"
	"github.com/forbearing/gst/service"
	"github.com/forbearing/gst/types"
)

type TestConnection struct {
	service.Base[*model.Empty, *modelaichat.Provider, *modelaichat.TestConnectionRsp]
}

type ListModels struct {
	service.Base[*model.Empty, *modelaichat.Provider, *modelaichat.ListModelsRsp]
}

// Create tests the connection to the AI provider
func (s *TestConnection) Create(ctx *types.ServiceContext, provider *modelaichat.Provider) (*modelaichat.TestConnectionRsp, error) {
	log := s.WithServiceContext(ctx, ctx.GetPhase())
	log.Infow("testing provider connection", "provider_id", provider.ID, "provider_type", provider.Type)

	// Create HTTP client with timeout
	httpClient := &http.Client{
		Timeout: 10 * time.Second,
	}

	// Get provider configuration
	config := provider.Config.Data()

	// Test connection based on provider type
	var success bool
	var message string

	switch provider.Type {
	case modelaichat.ProviderOpenAI, modelaichat.ProviderCustom:
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

	return &modelaichat.TestConnectionRsp{
		Success: success,
		Message: message,
	}, nil
}

// Create lists all models provided by the AI provider
func (s *ListModels) Create(ctx *types.ServiceContext, provider *modelaichat.Provider) (*modelaichat.ListModelsRsp, error) {
	log := s.WithServiceContext(ctx, ctx.GetPhase())
	log.Infow("listing provider models", "provider_id", provider.ID, "provider_type", provider.Type)

	// Create HTTP client with timeout
	httpClient := &http.Client{
		Timeout: 30 * time.Second,
	}

	// Get provider configuration
	config := provider.Config.Data()

	var models []modelaichat.ModelInfo

	switch provider.Type {
	case modelaichat.ProviderOpenAI, modelaichat.ProviderCustom:
		// List OpenAI-compatible models
		baseURL := config.BaseURL
		if baseURL == "" {
			baseURL = "https://api.openai.com/v1"
		}
		modelsURL := fmt.Sprintf("%s/models", baseURL)

		req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, modelsURL, nil)
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
			return nil, errors.New(fmt.Sprintf("failed to fetch models: status %d", resp.StatusCode))
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
			models = append(models, modelaichat.ModelInfo{
				ID:      m.ID,
				Name:    m.ID,
				Type:    "chat", // Default to chat for OpenAI models
				Context: 0,      // Context length not available in list endpoint
			})
		}

	case modelaichat.ProviderLocal:
		// List local models (Ollama)
		baseURL := config.BaseURL
		if baseURL == "" {
			baseURL = "http://localhost:11434"
		}
		modelsURL := fmt.Sprintf("%s/api/tags", baseURL)

		req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, modelsURL, nil)
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
			return nil, errors.New(fmt.Sprintf("failed to fetch models: status %d", resp.StatusCode))
		}

		var ollamaResp struct {
			Models []struct {
				Name       string `json:"name"`
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
			models = append(models, modelaichat.ModelInfo{
				ID:      m.Name,
				Name:    m.Name,
				Type:    "chat", // Default to chat for Ollama models
				Context: m.Details.ContextSize,
			})
		}

	default:
		// For other provider types, return empty list
		// They can be manually added or implemented later
		models = []modelaichat.ModelInfo{}
	}

	log.Infow("provider models listed", "count", len(models))

	return &modelaichat.ListModelsRsp{
		Models: models,
	}, nil
}
