package modelaichat

import (
	"github.com/forbearing/gst/model"
)

// Conversation represents a conversation session between a user and an AI model
type Conversation struct {
	UserID  string `gorm:"size:100;not null;index" json:"user_id,omitempty" schema:"user_id"`   // User ID
	Title   string `gorm:"size:200;not null" json:"title,omitempty" schema:"title"`             // Conversation title
	Summary string `gorm:"size:500" json:"summary,omitempty" schema:"summary"`                  // Conversation summary
	ModelID string `gorm:"size:100;not null;index" json:"model_id,omitempty" schema:"model_id"` // Model ID

	// Statistics
	MessageCount int `gorm:"default:0" json:"message_count,omitempty"` // Message count
	TokensUsed   int `gorm:"default:0" json:"tokens_used,omitempty"`   // Tokens used

	// Status
	IsArchived *bool `gorm:"default:false" json:"is_archived,omitempty" schema:"is_archived"` // Whether archived
	IsPinned   *bool `gorm:"default:false" json:"is_pinned,omitempty" schema:"is_pinned"`     // Whether pinned

	Model    *Model    `gorm:"-" json:"model,omitempty"`
	Messages []Message `gorm:"-" json:"messages,omitempty"`

	model.Base
}

func (*Conversation) Purge() bool          { return true }
func (*Conversation) GetTableName() string { return "ai_conversations" }
