package modelaichat

import "github.com/forbearing/gst/model"

// AgentFavorite represents a user's favorite agent
type AgentFavorite struct {
	UserID  string `gorm:"size:100;not null;index:idx_user_agent,unique" json:"user_id" schema:"user_id"`
	AgentID string `gorm:"size:100;not null;index:idx_user_agent,unique" json:"agent_id" schema:"agent_id"`

	Agent *Agent `gorm:"-" json:"agent,omitempty"`

	model.Base
}

func (AgentFavorite) Purge() bool          { return true }
func (AgentFavorite) GetTableName() string { return "ai_agent_favorites" }
