package modelaichat

// ChatCompletionReq represents a request for chat completion
type ChatCompletionReq struct {
	ConversationID string   `json:"conversation_id" schema:"conversation_id"` // Conversation ID (optional, create new if empty)
	ModelID        string   `json:"model_id" schema:"model_id"`               // Model ID
	Messages       []string `json:"messages" schema:"messages"`               // Message content (user messages)
	Stream         bool     `json:"stream,omitempty" schema:"stream"`         // Whether to stream response
}

// ChatCompletionRsp represents a response for chat completion
type ChatCompletionRsp struct {
	ConversationID string `json:"conversation_id"`   // Conversation ID
	MessageID      string `json:"message_id"`        // Created message ID
	Content        string `json:"content,omitempty"` // Response content (non-streaming only)
}

// StopMessageReq represents the request to stop a message
type StopMessageReq struct {
	MessageID string `json:"message_id" schema:"message_id"` // Message ID to stop
}

// StopMessageRsp represents the response to stop a message
type StopMessageRsp struct {
	MessageID string `json:"message_id"`
	Content   string `json:"content"`
}
