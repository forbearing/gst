package modelaichat

import (
	"time"

	"github.com/forbearing/gst/model"
)

// ConversationShare represents a shared conversation
type ConversationShare struct {
	ConversationID string     `gorm:"size:100;not null;index" json:"conversation_id" schema:"conversation_id"` // Conversation ID
	ShareCode      string     `gorm:"size:100;unique;not null;index" json:"share_code" schema:"share_code"`    // Share code
	ShareURL       string     `gorm:"size:500" json:"share_url,omitempty"`                                     // Share URL
	Password       *string    `gorm:"size:100" json:"password,omitempty"`                                      // Optional password protection
	ExpiresAt      *time.Time `json:"expires_at,omitempty"`                                                    // Expiration time
	ViewCount      int        `gorm:"default:0" json:"view_count,omitempty"`                                   // View count
	MaxViews       *int       `json:"max_views,omitempty"`                                                     // Maximum view count
	IsPublic       *bool      `gorm:"default:false" json:"is_public,omitempty" schema:"is_public"`             // Whether is public
	AllowCopy      *bool      `gorm:"default:true" json:"allow_copy,omitempty" schema:"allow_copy"`            // Whether to allow copying content

	model.Base
}

func (ConversationShare) Purge() bool          { return true }
func (ConversationShare) GetTableName() string { return "ai_conversation_shares" }
