package model

import (
	. "github.com/forbearing/gst/dsl"
	"github.com/forbearing/gst/model"
)

// ConversationType identifies the content type handled by a conversation.
type ConversationType string

const (
	ConversationTypeChat  ConversationType = "chat"
	ConversationTypeImage ConversationType = "image"
)

// Conversation demonstrates a database-backed resource with custom service hooks.
type Conversation struct {
	Type ConversationType `json:"type" schema:"type"`

	UserID string `json:"user_id" schema:"user_id"`
	Title  string `json:"title" schema:"title"`

	// Username is returned to clients and is not stored in the database.
	Username string `json:"username,omitempty" gorm:"-"`

	model.Base
}

func (Conversation) Design() {
	Migrate(true)
	Endpoint("conversations")
	Param("conv")

	Create(func() {
		Enabled(true)
		Service(true)
	})
	Delete(func() {
		Enabled(true)
		Service(true)
	})
	Patch(func() {
		Enabled(true)
		Service(true)
	})
	List(func() {
		Enabled(true)
		Service(true)
	})
	Get(func() {
		Enabled(true)
	})
}

func (Conversation) Purge() bool { return true }
