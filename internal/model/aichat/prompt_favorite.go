package modelaichat

import "github.com/forbearing/gst/model"

// PromptFavorite represents a user's favorite prompt
type PromptFavorite struct {
	UserID   string `gorm:"size:100;not null;index:idx_user_prompt,unique" json:"user_id" schema:"user_id"`
	PromptID string `gorm:"size:100;not null;index:idx_user_prompt,unique" json:"prompt_id" schema:"prompt_id"`

	Prompt *Prompt `gorm:"-" json:"prompt,omitempty"`

	model.Base
}

func (PromptFavorite) Purge() bool          { return true }
func (PromptFavorite) GetTableName() string { return "ai_prompt_favorites" }
