package modelaichat

import (
	"github.com/forbearing/gst/model"
	"gorm.io/datatypes"
)

// FeedbackType represents the type of user feedback
type FeedbackType string

const (
	FeedbackLike    FeedbackType = "like"    // Positive feedback
	FeedbackDislike FeedbackType = "dislike" // Negative feedback
)

// FeedbackCategory represents detailed feedback categories
type FeedbackCategory string

const (
	FeedbackCategoryInaccurate FeedbackCategory = "inaccurate"  // Inaccurate information
	FeedbackCategoryIncomplete FeedbackCategory = "incomplete"  // Incomplete answer
	FeedbackCategoryUnsafe     FeedbackCategory = "unsafe"       // Unsafe content
	FeedbackCategoryNotHelpful FeedbackCategory = "not_helpful" // Not helpful
	FeedbackCategoryTooVerbose FeedbackCategory = "too_verbose" // Too verbose
	FeedbackCategoryPoorFormat FeedbackCategory = "poor_format" // Poor formatting
	FeedbackCategoryOther      FeedbackCategory = "other"       // Other reasons
)

// MessageFeedback represents user feedback for a message
type MessageFeedback struct {
	MessageID string       `gorm:"size:100;uniqueIndex;not null" json:"message_id,omitempty" schema:"message_id"` // Message ID
	UserID    string       `gorm:"size:100;index;not null" json:"user_id,omitempty" schema:"user_id"`              // User ID
	Type      FeedbackType `gorm:"size:20;not null;" json:"type,omitempty" schema:"type"`                          // Feedback type

	// Detailed feedback (optional)
	Categories     datatypes.JSONSlice[FeedbackCategory] `gorm:"serializer:json" json:"categories,omitempty"` // Feedback categories
	Comment        string                                `gorm:"size:1000" json:"comment,omitempty"`            // User comment
	ExpectedAnswer string                                `gorm:"type:text" json:"expected_answer,omitempty"`    // Expected answer

	model.Base
}

func (MessageFeedback) Purge() bool          { return true }
func (MessageFeedback) GetTableName() string { return "ai_message_feedbacks" }
