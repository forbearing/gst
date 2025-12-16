package modelaichat

import "github.com/forbearing/gst/model"

// FavoriteType represents the type of favorited resource
type FavoriteType string

const (
	FavoriteTypePrompt FavoriteType = "prompt" // Prompt favorite
	FavoriteTypeAgent  FavoriteType = "agent"  // Agent favorite
)

// Favorite represents a user's favorite resource (prompt, agent, etc.)
type Favorite struct {
	UserID       string       `gorm:"size:100;not null;index:idx_user_resource,priority:1" json:"user_id" schema:"user_id"`                // User ID
	ResourceType FavoriteType `gorm:"size:20;not null;index:idx_user_resource,priority:2" json:"resource_type" schema:"resource_type"`     // Resource type
	ResourceID   string       `gorm:"size:100;not null;index:idx_user_resource,priority:3,unique" json:"resource_id" schema:"resource_id"` // Resource ID

	Prompt *Prompt `gorm:"-" json:"prompt,omitempty"`
	Agent  *Agent  `gorm:"-" json:"agent,omitempty"`

	model.Base
}

func (Favorite) Purge() bool          { return true }
func (Favorite) GetTableName() string { return "ai_favorites" }
