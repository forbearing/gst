package modelaichat

// ChatCompletionReq represents a request for chat completion
type ChatCompletionReq struct {
	ConversationID string   `json:"conversation_id"`  // Conversation ID (optional, create new if empty)
	ModelID        string   `json:"model_id"`         // Model ID
	Messages       []string `json:"messages"`         // Message content (user messages)
	Stream         bool     `json:"stream,omitempty"` // Whether to stream response
}

// ChatCompletionRsp represents a response for chat completion
type ChatCompletionRsp struct {
	ConversationID string `json:"conversation_id,omitempty"` // Conversation ID
	MessageID      string `json:"message_id,omitempty"`      // Created message ID
	Content        string `json:"content,omitempty"`         // Response content (non-streaming only)
}

// StopMessageReq represents the request to stop a message
type StopMessageReq struct {
	MessageID string `json:"message_id"` // Message ID to stop
}

// StopMessageRsp represents the response to stop a message
type StopMessageRsp struct {
	MessageID string `json:"message_id,omitempty"`
	Content   string `json:"content,omitempty"`
}

// RegenerateMessageReq represents a request to regenerate an assistant message
type RegenerateMessageReq struct {
	MessageID string `json:"message_id"`       // Assistant message ID to regenerate
	Stream    bool   `json:"stream,omitempty"` // Whether to stream response
}

// RegenerateMessageRsp represents a response for regenerated message
type RegenerateMessageRsp struct {
	MessageID      string `json:"message_id,omitempty"`      // New regenerated message ID
	ConversationID string `json:"conversation_id,omitempty"` // Conversation ID
	Content        string `json:"content,omitempty"`         // Response content (non-streaming only)
}

// SubmitMessageFeedbackReq represents a request to submit feedback for a message
type SubmitMessageFeedbackReq struct {
	MessageID      string             `json:"message_id"`                // Message ID to provide feedback for
	Type           FeedbackType       `json:"type"`                      // Feedback type (like/dislike)
	Categories     []FeedbackCategory `json:"categories,omitempty"`      // Detailed feedback categories (optional)
	Comment        string             `json:"comment,omitempty"`         // User comment (optional)
	ExpectedAnswer string             `json:"expected_answer,omitempty"` // Expected answer (optional)
}

// SubmitMessageFeedbackRsp represents a response for submitted feedback
type SubmitMessageFeedbackRsp struct {
	FeedbackID string `json:"feedback_id,omitempty"` // Created or updated feedback ID
	MessageID  string `json:"message_id,omitempty"`  // Message ID
}

// ClearConversationMessagesReq represents a request to clear all messages in a conversation
type ClearConversationMessagesReq struct {
	ConversationID string `json:"conversation_id"` // Conversation ID to clear messages from
}

// ClearConversationMessagesRsp represents a response for clearing conversation messages
type ClearConversationMessagesRsp struct {
	ConversationID string `json:"conversation_id,omitempty"` // Conversation ID
	DeletedCount   int    `json:"deleted_count,omitempty"`   // Number of messages deleted
}
