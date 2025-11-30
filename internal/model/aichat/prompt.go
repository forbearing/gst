package modelaichat

import (
	"github.com/forbearing/gst/model"
	"gorm.io/datatypes"
)

// PromptType represents the type of prompt
type PromptType string

const (
	PromptTypeSystem    PromptType = "system"    // System prompt
	PromptTypeUser      PromptType = "user"      // User prompt template
	PromptTypeAssistant PromptType = "assistant" // Assistant prompt template
)

// PromptCategory represents the category of prompt
type PromptCategory string

const (
	PromptCategoryGeneral     PromptCategory = "general"     // General purpose
	PromptCategoryWriting     PromptCategory = "writing"     // Writing assistance
	PromptCategoryCoding      PromptCategory = "coding"      // Code generation
	PromptCategoryAnalysis    PromptCategory = "analysis"    // Data analysis
	PromptCategoryTranslation PromptCategory = "translation" // Translation
	PromptCategoryChat        PromptCategory = "chat"        // Chat/Conversation
	PromptCategoryRAG         PromptCategory = "rag"         // RAG specific
	PromptCategoryAgent       PromptCategory = "agent"       // Agent specific
	PromptCategoryCustom      PromptCategory = "custom"      // Custom category
)

// PromptVisibility represents the visibility of prompt
type PromptVisibility string

const (
	PromptVisibilityPrivate PromptVisibility = "private" // Only visible to owner
	PromptVisibilityTeam    PromptVisibility = "team"    // Visible to team members
	PromptVisibilityPublic  PromptVisibility = "public"  // Visible to all users
)

// Prompt represents a prompt template
type Prompt struct {
	Name        string           `gorm:"size:100;not null" json:"name" schema:"name"`
	Description string           `gorm:"size:500" json:"description,omitempty" schema:"description"`
	Content     string           `gorm:"type:text;not null" json:"content"`
	Type        PromptType       `gorm:"size:20;default:system" json:"type,omitempty" schema:"type"`
	Category    PromptCategory   `gorm:"size:30;default:general" json:"category,omitempty" schema:"category"`
	Visibility  PromptVisibility `gorm:"size:20;default:private" json:"visibility,omitempty" schema:"visibility"`
	UserID      string           `gorm:"size:100;not null;index" json:"user_id,omitempty" schema:"user_id"` // Owner user ID

	Variables datatypes.JSONSlice[string] `gorm:"size:1000" json:"variables,omitempty"` // JSON array of variable names

	// Usage statistics
	UseCount      int   `gorm:"default:0" json:"use_count,omitempty"`      // Usage count
	LastUsedAt    int64 `json:"last_used_at,omitempty"`                    // Last used timestamp
	FavoriteCount int   `gorm:"default:0" json:"favorite_count,omitempty"` // Favorite count

	model.Base
}

func (Prompt) Purge() bool          { return true }
func (Prompt) GetTableName() string { return "ai_prompts" }
