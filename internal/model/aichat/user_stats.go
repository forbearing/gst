package modelaichat

import (
	"time"

	"github.com/forbearing/gst/model"
	"gorm.io/datatypes"
)

// UsageStats represents daily usage statistics for a user
type UsageStats struct {
	UserID string    `gorm:"size:100;not null;index:idx_user_date,priority:1" json:"user_id" schema:"user_id"` // User ID
	Date   time.Time `gorm:"type:date;not null;index:idx_user_date,priority:2" json:"date" schema:"date"`      // Statistics date

	// Token statistics
	PromptTokens     int64 `gorm:"default:0" json:"prompt_tokens,omitempty"`     // Prompt tokens used
	CompletionTokens int64 `gorm:"default:0" json:"completion_tokens,omitempty"` // Completion tokens used
	TotalTokens      int64 `gorm:"default:0" json:"total_tokens,omitempty"`      // Total tokens used

	// Request statistics
	RequestCount      int `gorm:"default:0" json:"request_count,omitempty"`      // Request count
	ConversationCount int `gorm:"default:0" json:"conversation_count,omitempty"` // Conversation count
	MessageCount      int `gorm:"default:0" json:"message_count,omitempty"`      // Message count

	// Model statistics (JSON)
	ModelStats datatypes.JSON `json:"model_stats,omitempty"` // Statistics by model (JSON)

	model.Base
}

func (UsageStats) Purge() bool          { return true }
func (UsageStats) GetTableName() string { return "ai_usage_stats" }
