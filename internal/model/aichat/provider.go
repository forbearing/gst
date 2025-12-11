package modelaichat

import (
	"github.com/forbearing/gst/model"
	"gorm.io/datatypes"
)

// ProviderType represents the type of AI provider
type ProviderType string

const (
	ProviderOpenAI    ProviderType = "openai"
	ProviderAnthropic ProviderType = "anthropic"
	ProviderGoogle    ProviderType = "google"
	ProviderAzure     ProviderType = "azure"
	ProviderAliyun    ProviderType = "aliyun" // Alibaba Cloud Tongyi
	ProviderBaidu     ProviderType = "baidu"  // Baidu Wenxin
	ProviderLocal     ProviderType = "local"  // Local deployment (Ollama, etc.)
	ProviderCustom    ProviderType = "custom" // Custom OpenAI-compatible API
)

// ProviderConfig stores sensitive provider configuration
type ProviderConfig struct {
	APIKey       string            `json:"api_key,omitempty"`
	SecretKey    string            `json:"secret_key,omitempty"`
	BaseURL      string            `json:"base_url,omitempty"`
	OrgID        string            `json:"org_id,omitempty"`
	APIVersion   string            `json:"api_version,omitempty"`
	Region       string            `json:"region,omitempty"`
	ExtraHeaders map[string]string `json:"extra_headers,omitempty"`
	ExtraParams  map[string]string `json:"extra_params,omitempty"`
}

// Provider represents an AI provider
type Provider struct {
	Name string       `gorm:"size:100;not null;uniqueIndex" json:"name" schema:"name"` // Display name
	Code string       `gorm:"size:50;not null;uniqueIndex" json:"code" schema:"code"`  // Unique identifier
	Type ProviderType `gorm:"size:20;not null;index" json:"type" schema:"type"`        // Provider type

	Config datatypes.JSONType[ProviderConfig] `json:"config"` // AI provider configuration

	Description string `gorm:"size:500" json:"description"`
	Icon        string `gorm:"size:255" json:"icon"`    // Icon URL
	Status      *int   `gorm:"default:1" json:"status"` // Status: 1 enabled, 0 disabled

	Models []Model `gorm:"-" json:"models,omitempty"`

	model.Base
}

func (Provider) Purge() bool          { return true }
func (Provider) GetTableName() string { return "ai_providers" }

// ProviderSyncModelsRsp is the response type for syncing provider models into database
type ProviderSyncModelsRsp struct {
	Total   int `json:"total"`   // Total models fetched from provider
	Created int `json:"created"` // Number of newly created models
	Updated int `json:"updated"` // Number of updated models
	Failed  int `json:"failed"`  // Number of failed models
}

// ProviderTestConnRsp is the response type for testing provider connection
type ProviderTestConnRsp struct {
	Success   bool     `json:"success"`
	Message   string   `json:"message,omitempty"`
	Latency   int64    `json:"latency,omitempty"` // milliseconds
	ModelList []string `json:"model_list,omitempty"`
}
