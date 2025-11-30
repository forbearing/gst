package modelaichat

import "github.com/forbearing/gst/model"

// MessageRole represents the role of a message in a conversation
type MessageRole string

const (
	MessageRoleSystem    MessageRole = "system"    // System message
	MessageRoleUser      MessageRole = "user"      // User message
	MessageRoleAssistant MessageRole = "assistant" // Assistant message
)

// MessageStatus represents the status of a message
type MessageStatus string

const (
	MessageStatusPending   MessageStatus = "pending"   // Pending processing
	MessageStatusStreaming MessageStatus = "streaming" // Streaming generation
	MessageStatusCompleted MessageStatus = "completed" // Completed
	MessageStatusStopped   MessageStatus = "stopped"   // User stopped
	MessageStatusFailed    MessageStatus = "failed"    // Generation failed
)

// StopReason represents the reason why message generation stopped
type StopReason string

const (
	StopReasonUser      StopReason = "user"       // User stopped
	StopReasonMaxTokens StopReason = "max_tokens" // Reached max tokens
	StopReasonTimeout   StopReason = "timeout"    // Timeout
	StopReasonError     StopReason = "error"      // Error occurred
	StopReasonEndTurn   StopReason = "end_turn"   // Model ended normally
)

// Message represents a single message in a chat conversation
type Message struct {
	Role       MessageRole   `gorm:"size:20;not null;index" json:"role" schema:"role"`                           // Message role
	Status     MessageStatus `gorm:"size:20;default:completed" json:"status,omitempty" schema:"status"`         // Message status
	Content    string        `gorm:"type:text" json:"content"`                                                    // Message content
	ErrMessage string        `gorm:"type:text" json:"err_message,omitempty"`                                     // Error message if failed
	StopReason *StopReason   `gorm:"size:20" json:"stop_reason,omitempty" schema:"stop_reason"`                  // Stop reason

	// Model information
	ParentID   *string          `gorm:"index" json:"parent_id,omitempty" schema:"parent_id"` // Parent message ID (for regeneration versioning)
	ChatID     string           `gorm:"index;not null" json:"chat_id" schema:"chat_id"`      // Chat ID
	ModelID    string           `gorm:"index;not null" json:"model_id" schema:"model_id"`    // Model ID
	Chat       *Chat            `gorm:"-" json:"chat,omitempty"`
	Model      *Model           `gorm:"-" json:"model,omitempty"`
	Variations []Message        `gorm:"-" json:"variations,omitempty"`
	Feedback   *MessageFeedback `gorm:"-" json:"feedback,omitempty"`

	// Token statistics
	PromptTokens     int `gorm:"default:0" json:"prompt_tokens,omitempty"`     // Input tokens
	CompletionTokens int `gorm:"default:0" json:"completion_tokens,omitempty"` // Output tokens
	TotalTokens      int `gorm:"default:0" json:"total_tokens,omitempty"`      // Total tokens

	// Regeneration related
	RegenerateCount int  `gorm:"default:0" json:"regenerate_count,omitempty"` // Regeneration count
	IsActive        bool `gorm:"default:true" json:"is_active,omitempty"`     // Whether is active version

	// Performance
	LatencyMs int64 `json:"latency_ms,omitempty"` // Response latency in milliseconds

	model.Base
}

func (Message) Purge() bool          { return true }
func (Message) GetTableName() string { return "ai_messages" }
