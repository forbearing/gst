package modelaichat

import (
	"github.com/forbearing/gst/model"
)

// AgentTool represents a tool/function that an agent can use
type AgentTool struct {
	Name        string `gorm:"size:100;not null" json:"name" schema:"name"`                // Tool name
	Description string `gorm:"size:500" json:"description,omitempty" schema:"description"` // Description
	Type        string `gorm:"size:50;not null" json:"type" schema:"type"`                 // Tool type (function, api, plugin)

	UserID string `gorm:"size:100;not null;index" json:"user_id,omitempty" schema:"user_id"` // Owner user ID

	// Function definition (OpenAI function calling format)
	FunctionDef string `gorm:"type:text" json:"function_def,omitempty" schema:"function_def"` // JSON function definition

	// API configuration (for API type tools)
	APIEndpoint string `gorm:"size:500" json:"api_endpoint,omitempty" schema:"api_endpoint"` // API endpoint
	APIMethod   string `gorm:"size:10" json:"api_method,omitempty" schema:"api_method"`      // HTTP method
	APIHeaders  string `gorm:"size:1000" json:"api_headers,omitempty" schema:"api_headers"`  // JSON headers
	APIAuth     string `gorm:"size:500" json:"api_auth,omitempty" schema:"api_auth"`         // JSON auth config

	// Status
	Enabled *bool `gorm:"default:true" json:"enabled,omitempty" schema:"enabled"` // Is enabled

	model.Base
}

func (AgentTool) Purge() bool          { return true }
func (AgentTool) GetTableName() string { return "ai_agent_tools" }
