package modelaichat

import (
	"github.com/forbearing/gst/model"
	"gorm.io/datatypes"
)

// AgentStatus represents the status of an agent
type AgentStatus string

const (
	AgentStatusDraft     AgentStatus = "draft"     // Draft, not published
	AgentStatusPublished AgentStatus = "published" // Published and available
	AgentStatusDisabled  AgentStatus = "disabled"  // Disabled
	AgentStatusArchived  AgentStatus = "archived"  // Archived
)

// AgentVisibility represents the visibility of an agent
type AgentVisibility string

const (
	AgentVisibilityPrivate AgentVisibility = "private" // Only visible to owner
	AgentVisibilityTeam    AgentVisibility = "team"    // Visible to team members
	AgentVisibilityPublic  AgentVisibility = "public"  // Visible to all users
)

// AgentCategory represents the category of an agent
type AgentCategory string

const (
	AgentCategoryAssistant       AgentCategory = "assistant"        // General assistant
	AgentCategoryCustomerService AgentCategory = "customer_service" // Customer service
	AgentCategoryCoding          AgentCategory = "coding"           // Coding assistant
	AgentCategoryWriting         AgentCategory = "writing"          // Writing assistant
	AgentCategoryAnalysis        AgentCategory = "analysis"         // Data analysis
	AgentCategoryEducation       AgentCategory = "education"        // Education/Tutoring
	AgentCategoryCreative        AgentCategory = "creative"         // Creative/Design
	AgentCategoryProductivity    AgentCategory = "productivity"     // Productivity tools
	AgentCategoryCustom          AgentCategory = "custom"           // Custom category
)

type Agent struct {
	Name        string          `gorm:"size:100;not null" json:"name" schema:"name"`                             // Agent name
	Description string          `gorm:"size:1000" json:"description,omitempty" schema:"description"`             // Description
	Avatar      string          `gorm:"size:500" json:"avatar,omitempty" schema:"avatar"`                        // Avatar URL
	Status      AgentStatus     `gorm:"size:20;default:draft" json:"status,omitempty" schema:"status"`           // Status
	Visibility  AgentVisibility `gorm:"size:20;default:private" json:"visibility,omitempty" schema:"visibility"` // Visibility
	Category    AgentCategory   `gorm:"size:30;default:assistant" json:"category,omitempty" schema:"category"`   // Category

	UserID     string `gorm:"size:100;not null;index" json:"user_id,omitempty" schema:"user_id"` // Owner user ID
	ProviderID string `gorm:"size:100;index" json:"provider_id,omitempty" schema:"provider_id"`  // Associated provider ID
	ModelID    string `gorm:"size:100;index" json:"model_id,omitempty" schema:"model_id"`        // Associated model ID
	ChatID     string `gorm:"size:100;index" json:"chat_id,omitempty" schema:"chat_id"`          // Associated chat ID
	PromptID   string `gorm:"size:100;index" json:"prompt_id,omitempty" schema:"prompt_id"`      // Associated prompt ID

	// Knowledge base configuration
	EnableRAG        *bool                       `gorm:"default:false" json:"enable_rag,omitempty" schema:"enable_rag"`         // Enable RAG
	KnowledgeBaseIDs datatypes.JSONSlice[string] `gorm:"size:1000" json:"knowledge_base_ids,omitempty"`                         // JSON array of knowledge base IDs
	TopK             int                         `gorm:"default:5" json:"top_k,omitempty" schema:"top_k"`                       // Number of chunks to retrieve
	ScoreThreshold   float64                     `gorm:"default:0.5" json:"score_threshold,omitempty" schema:"score_threshold"` // Similarity score threshold

	// Tool configuration
	EnableTools bool                        `gorm:"default:false" json:"enable_tools,omitempty" schema:"enable_tools"` // Enable tools/functions
	ToolIDs     datatypes.JSONSlice[string] `gorm:"size:1000" json:"tool_ids,omitempty" schema:"tool_ids"`             // JSON array of tool IDs

	Prompt         *Prompt          `gorm:"-" json:"prompt,omitempty"`          // Associated prompt
	Provider       *Provider        `gorm:"-" json:"provider,omitempty"`        // Associated provider
	Model          *Model           `gorm:"-" json:"model,omitempty"`           // Associated model
	Conversation   *Conversation    `gorm:"-" json:"conversation,omitempty"`    // Associated conversation
	KnowledgeBases []*KnowledgeBase `gorm:"-" json:"knowledge_bases,omitempty"` // Associated knowledge bases
	Tools          []*AgentTool     `gorm:"-" json:"tools,omitempty"`           // Associated tools

	model.Base
}

func (Agent) Purge() bool          { return true }
func (Agent) GetTableName() string { return "ai_agents" }
