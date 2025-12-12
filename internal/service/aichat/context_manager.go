package serviceaichat

import (
	"strings"

	"github.com/cloudwego/eino/schema"
	"github.com/cockroachdb/errors"
	modelaichat "github.com/forbearing/gst/internal/model/aichat"
	tiktoken "github.com/pkoukk/tiktoken-go"
)

// ContextManager manages conversation context window
type ContextManager struct {
	modelID       string
	contextLength int
	encoding      *tiktoken.Tiktoken
}

// NewContextManager creates a new context manager
func NewContextManager(modelID string, contextLength int) (*ContextManager, error) {
	if contextLength <= 0 {
		// Default context length if not specified
		contextLength = 4096
	}

	// Get encoding for the model
	encoding, err := getEncodingForModel(modelID)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get encoding for model: %s", modelID)
	}

	return &ContextManager{
		modelID:       modelID,
		contextLength: contextLength,
		encoding:      encoding,
	}, nil
}

// getEncodingForModel returns the appropriate tiktoken encoding for a given model
// It uses prefix matching to support new models from OpenAI and Anthropic
// For unknown models, it first tries to use the model ID directly, then falls back to cl100k_base
func getEncodingForModel(modelID string) (*tiktoken.Tiktoken, error) {
	// Try to get encoding using the model ID directly first
	// This works for most OpenAI models as tiktoken supports model name lookup
	if encoding, err := tiktoken.EncodingForModel(modelID); err == nil {
		return encoding, nil
	}

	// Fallback to prefix-based matching for better compatibility
	// OpenAI models: gpt-4*, gpt-3.5-turbo*, gpt-4o*, o1*, etc.
	if strings.HasPrefix(modelID, "gpt-4o") {
		return tiktoken.EncodingForModel("gpt-4o")
	}
	if strings.HasPrefix(modelID, "gpt-4") {
		return tiktoken.EncodingForModel("gpt-4")
	}
	if strings.HasPrefix(modelID, "gpt-3.5-turbo") {
		return tiktoken.EncodingForModel("gpt-3.5-turbo")
	}
	if strings.HasPrefix(modelID, "o1") {
		// o1 models use o200k_base encoding
		return tiktoken.GetEncoding("o200k_base")
	}

	// Anthropic Claude models: claude-*, use cl100k_base (same as GPT-3.5/4)
	if strings.HasPrefix(modelID, "claude-") {
		return tiktoken.EncodingForModel("gpt-3.5-turbo")
	}

	// Default to cl100k_base for all unknown models
	// This ensures context window management works for all models
	// Most modern LLMs use cl100k_base or compatible encodings
	return tiktoken.GetEncoding("cl100k_base")
}

// CountTokens counts the number of tokens in a message
func (cm *ContextManager) CountTokens(msg *schema.Message) int {
	if msg == nil {
		return 0
	}

	// Count tokens using tiktoken
	tokens := cm.encoding.Encode(msg.Content, nil, nil)
	// Add overhead for message formatting (approximately 4 tokens per message)
	// The overhead accounts for role tags and message structure
	return len(tokens) + 4
}

// CountTokensForMessages counts total tokens for a slice of messages
func (cm *ContextManager) CountTokensForMessages(messages []*schema.Message) int {
	total := 0
	for _, msg := range messages {
		total += cm.CountTokens(msg)
	}
	return total
}

// ManageContext manages the context window by trimming messages if necessary
// It preserves system messages and uses a sliding window strategy for conversation history
// historyMessages: all historical messages from database (user messages + assistant replies)
// newUserMessages: new user messages from current request
// Returns trimmed messages that fit within the context window
func (cm *ContextManager) ManageContext(
	historyMessages []*modelaichat.Message,
	newUserMessages []string,
) ([]*schema.Message, error) {
	// Reserve 20% of context length for new messages and response
	reservedTokens := int(float64(cm.contextLength) * 0.2)
	availableTokens := cm.contextLength - reservedTokens

	// Build initial eino messages from history
	einoMessages := make([]*schema.Message, 0)

	// Separate system messages from conversation messages
	systemMessages := make([]*schema.Message, 0)
	conversationMessages := make([]*schema.Message, 0)

	// Convert history messages to eino messages
	for _, msg := range historyMessages {
		var einoMsg *schema.Message
		switch msg.Role {
		case modelaichat.MessageRoleSystem:
			einoMsg = schema.SystemMessage(msg.Content)
			systemMessages = append(systemMessages, einoMsg)
		case modelaichat.MessageRoleUser:
			einoMsg = schema.UserMessage(msg.Content)
			conversationMessages = append(conversationMessages, einoMsg)
		case modelaichat.MessageRoleAssistant:
			einoMsg = schema.AssistantMessage(msg.Content, nil)
			conversationMessages = append(conversationMessages, einoMsg)
		}
	}

	// Add new user messages to conversation
	for _, content := range newUserMessages {
		conversationMessages = append(conversationMessages, schema.UserMessage(content))
	}

	// Count tokens for system messages
	systemTokens := cm.CountTokensForMessages(systemMessages)

	// Calculate available tokens for conversation (excluding system messages)
	conversationAvailableTokens := max(0, availableTokens-systemTokens)

	// Trim conversation messages if necessary using sliding window
	trimmedConversationMessages := cm.trimMessages(conversationMessages, conversationAvailableTokens)

	// Combine system messages and trimmed conversation messages
	einoMessages = append(einoMessages, systemMessages...)
	einoMessages = append(einoMessages, trimmedConversationMessages...)

	return einoMessages, nil
}

// trimMessages trims messages using a sliding window strategy
// It keeps the most recent messages that fit within the token limit
func (cm *ContextManager) trimMessages(messages []*schema.Message, maxTokens int) []*schema.Message {
	if len(messages) == 0 {
		return messages
	}

	// If all messages fit, return them as is
	totalTokens := cm.CountTokensForMessages(messages)
	if totalTokens <= maxTokens {
		return messages
	}

	// Use sliding window: keep the most recent messages
	// Start from the end and work backwards
	trimmed := make([]*schema.Message, 0)
	currentTokens := 0

	// Iterate backwards through messages
	for i := len(messages) - 1; i >= 0; i-- {
		msgTokens := cm.CountTokens(messages[i])
		if currentTokens+msgTokens > maxTokens {
			// Can't fit this message, stop
			break
		}
		trimmed = append([]*schema.Message{messages[i]}, trimmed...)
		currentTokens += msgTokens
	}

	// If we have user messages, try to keep at least one user-assistant pair
	// This ensures we don't lose the conversation context completely
	if len(trimmed) == 0 && len(messages) > 0 {
		// At least keep the last message
		trimmed = []*schema.Message{messages[len(messages)-1]}
	}

	return trimmed
}
