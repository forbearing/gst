package modelaichat

import (
	"github.com/forbearing/gst/model"
	"gorm.io/datatypes"
)

type ModelType string

const (
	ModelTypeChat      ModelType = "chat"      // Chat model
	ModelTypeEmbedding ModelType = "embedding" // Embedding model
	ModelTypeImage     ModelType = "image"     // Image generation model
	ModelTypeAudio     ModelType = "audio"     // Audio model
	ModelTypeVision    ModelType = "vision"    // Vision understanding model
)

type ModelConfig struct {
	MaxTokens        int     `json:"max_tokens,omitempty"`        // Maximum output tokens
	ContextLength    int     `json:"context_length,omitempty"`    // Context length
	Temperature      float64 `json:"temperature,omitempty"`       // Default temperature
	TopP             float64 `json:"top_p,omitempty"`             // Default top_p
	FrequencyPenalty float64 `json:"frequency_penalty,omitempty"` // Frequency penalty
	PresencePenalty  float64 `json:"presence_penalty,omitempty"`  // Presence penalty
	SupportTools     bool    `json:"support_tools,omitempty"`     // Whether supports tool calling
	SupportVision    bool    `json:"support_vision,omitempty"`    // Whether supports vision
	SupportStreaming bool    `json:"support_streaming,omitempty"` // Whether supports streaming
}

type Model struct {
	ProviderID string    `gorm:"not null;index" json:"provider_id" schema:"provider_id"` // Provider ID
	Name       string    `gorm:"size:100;not null" json:"name" schema:"name"`            // Display name
	ModelID    string    `gorm:"size:100;not null" json:"model_id" schema:"model_id"`    // Model identifier (e.g., gpt-4o)
	Type       ModelType `gorm:"size:20;not null;index" json:"type" schema:"type"`       // Model type

	Config      datatypes.JSONType[ModelConfig] `json:"config"`                                           // AI model configuration
	InputPrice  float64                         `gorm:"type:decimal(10,6);default:0" json:"input_price"`  // Input price (per 1K tokens)
	OutputPrice float64                         `gorm:"type:decimal(10,6);default:0" json:"output_price"` // Output price (per 1K tokens)

	Description string `gorm:"size:500" json:"description,omitempty"`                         // Description
	IsDefault   *bool  `gorm:"default:false" json:"is_default,omitempty" schema:"is_default"` // Whether is default model
	Status      *int   `gorm:"default:1;index" json:"status" schema:"status"`                 // Status: 1 enabled, 0 disabled

	// // Association
	// Provider *Provider `gorm:"foreignKey:ProviderID" json:"provider,omitempty" schema:"provider"`

	model.Base
}

func (Model) Purge() bool          { return true }
func (Model) GetTableName() string { return "ai_models" }
