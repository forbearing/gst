package serviceaichat

import (
	"regexp"
	"strings"
	"unicode"

	"github.com/cockroachdb/errors"

	"github.com/forbearing/gst/database"
	modelaichat "github.com/forbearing/gst/internal/model/aichat"
	"github.com/forbearing/gst/types"
	"github.com/forbearing/gst/util"
)

const (
	// DefaultConversationTitle is the default title for new conversations
	DefaultConversationTitle = "New Conversation"
	// MaxTitleLength is the maximum length for auto-generated titles
	MaxTitleLength = 50
)

// GenerateTitleFromFirstMessage generates a conversation title from the first user message.
// It extracts the first meaningful content from the message and truncates it to MaxTitleLength.
func GenerateTitleFromFirstMessage(firstMessage string) string {
	if len(firstMessage) == 0 {
		return DefaultConversationTitle
	}

	// Remove leading/trailing whitespace
	title := strings.TrimSpace(firstMessage)

	// Replace newlines with space
	title = strings.ReplaceAll(title, "\n", " ")
	title = strings.ReplaceAll(title, "\r", " ")

	// Replace multiple spaces (including tabs) with single space using regex
	spaceRegex := regexp.MustCompile(`\s+`)
	title = spaceRegex.ReplaceAllString(title, " ")

	// Truncate to max length
	if len(title) > MaxTitleLength {
		// Try to truncate at word boundary
		truncated := title[:MaxTitleLength]
		lastSpace := strings.LastIndex(truncated, " ")
		if lastSpace > MaxTitleLength/2 {
			// If we found a space in the second half, truncate there
			title = truncated[:lastSpace] + "..."
		} else {
			// Otherwise, just truncate and add ellipsis
			title = truncated + "..."
		}
	}

	// Remove any trailing punctuation or spaces
	title = strings.TrimRightFunc(title, func(r rune) bool {
		return unicode.IsPunct(r) || unicode.IsSpace(r)
	})

	if len(title) == 0 {
		return DefaultConversationTitle
	}

	return title
}

// UpdateConversationTitleIfNeeded updates the conversation title if it's still the default title.
// It generates a title from the first user message in the conversation.
func UpdateConversationTitleIfNeeded(ctx *types.ServiceContext, conversationID string) error {
	// Get the conversation
	conversation := new(modelaichat.Conversation)
	if err := database.Database[*modelaichat.Conversation](ctx.DatabaseContext()).Get(conversation, conversationID); err != nil {
		return errors.Wrapf(err, "failed to get conversation: %s", conversationID)
	}

	// Only update if title is still the default
	if conversation.Title != DefaultConversationTitle {
		return nil
	}

	// Get user messages in this conversation, ordered by creation time
	// Try to find the first message that can generate a valid title
	// This handles cases where the first message might be empty or invalid
	query := &modelaichat.Message{
		ConversationID: conversationID,
		Role:           modelaichat.MessageRoleUser,
		Status:         modelaichat.MessageStatusCompleted,
		IsActive:       util.ValueOf(true),
	}
	userMessages := make([]*modelaichat.Message, 0)
	if err := database.Database[*modelaichat.Message](ctx.DatabaseContext()).
		WithQuery(query).
		WithOrder("created_at ASC").
		WithLimit(5). // Limit to first 5 messages to avoid excessive queries
		List(&userMessages); err != nil {
		return errors.Wrap(err, "failed to get user messages")
	}

	// If no user message found, keep default title
	if len(userMessages) == 0 {
		return nil
	}

	// Try to find the first message that can generate a valid title
	var newTitle string
	for _, msg := range userMessages {
		// Skip empty messages
		if len(msg.Content) == 0 {
			continue
		}

		// Generate title from this message
		title := GenerateTitleFromFirstMessage(msg.Content)

		// If we got a valid title (not default), use it
		if title != DefaultConversationTitle {
			newTitle = title
			break
		}
	}

	// If no valid title found, keep default title
	if newTitle == "" || newTitle == DefaultConversationTitle {
		return nil
	}

	// Update conversation title
	// Note: There's a potential race condition if multiple requests update the title simultaneously,
	// but this is acceptable since the function is only called when messages complete,
	// and conversations are typically processed sequentially.
	conversation.Title = newTitle
	if err := database.Database[*modelaichat.Conversation](ctx.DatabaseContext()).Update(conversation); err != nil {
		return errors.Wrap(err, "failed to update conversation title")
	}

	return nil
}
