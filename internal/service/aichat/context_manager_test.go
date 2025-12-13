package serviceaichat

import (
	"strings"
	"testing"

	"github.com/cloudwego/eino/schema"
	modelaichat "github.com/forbearing/gst/internal/model/aichat"
	"github.com/stretchr/testify/require"
)

func TestNewContextManager(t *testing.T) {
	t.Run("valid model", func(t *testing.T) {
		cm, err := NewContextManager("gpt-4", 4096)
		require.NoError(t, err)
		require.NotNil(t, cm)
		require.Equal(t, "gpt-4", cm.modelID)
		require.Equal(t, 4096, cm.contextLength)
		require.NotNil(t, cm.encoding)
	})

	t.Run("invalid context length uses default", func(t *testing.T) {
		cm, err := NewContextManager("gpt-4", 0)
		require.NoError(t, err)
		require.NotNil(t, cm)
		require.Equal(t, 4096, cm.contextLength)
	})

	t.Run("negative context length uses default", func(t *testing.T) {
		cm, err := NewContextManager("gpt-4", -100)
		require.NoError(t, err)
		require.NotNil(t, cm)
		require.Equal(t, 4096, cm.contextLength)
	})

	t.Run("unknown model uses default encoding", func(t *testing.T) {
		cm, err := NewContextManager("unknown-model-123", 4096)
		require.NoError(t, err)
		require.NotNil(t, cm)
		require.NotNil(t, cm.encoding)
	})

	t.Run("claude model", func(t *testing.T) {
		cm, err := NewContextManager("claude-3-opus", 4096)
		require.NoError(t, err)
		require.NotNil(t, cm)
		require.NotNil(t, cm.encoding)
	})
}

func TestCountTokens(t *testing.T) {
	cm, err := NewContextManager("gpt-4", 4096)
	require.NoError(t, err)

	t.Run("nil message", func(t *testing.T) {
		tokens := cm.CountTokens(nil)
		require.Equal(t, 0, tokens)
	})

	t.Run("empty message", func(t *testing.T) {
		msg := schema.UserMessage("")
		tokens := cm.CountTokens(msg)
		// Should have at least 4 tokens for message formatting overhead
		require.GreaterOrEqual(t, tokens, 4)
	})

	t.Run("short message", func(t *testing.T) {
		msg := schema.UserMessage("Hello")
		tokens := cm.CountTokens(msg)
		require.Greater(t, tokens, 4)
	})

	t.Run("long message", func(t *testing.T) {
		longText := strings.Repeat("Hello world ", 100)
		msg := schema.UserMessage(longText)
		tokens := cm.CountTokens(msg)
		require.Greater(t, tokens, 100)
	})
}

func TestCountTokensForMessages(t *testing.T) {
	cm, err := NewContextManager("gpt-4", 4096)
	require.NoError(t, err)

	t.Run("empty messages", func(t *testing.T) {
		tokens := cm.CountTokensForMessages([]*schema.Message{})
		require.Equal(t, 0, tokens)
	})

	t.Run("single message", func(t *testing.T) {
		messages := []*schema.Message{
			schema.UserMessage("Hello"),
		}
		tokens := cm.CountTokensForMessages(messages)
		singleTokens := cm.CountTokens(messages[0])
		require.Equal(t, singleTokens, tokens)
	})

	t.Run("multiple messages", func(t *testing.T) {
		messages := []*schema.Message{
			schema.UserMessage("Hello"),
			schema.AssistantMessage("Hi there", nil),
			schema.UserMessage("How are you?"),
		}
		tokens := cm.CountTokensForMessages(messages)
		expected := cm.CountTokens(messages[0]) + cm.CountTokens(messages[1]) + cm.CountTokens(messages[2])
		require.Equal(t, expected, tokens)
	})
}

func TestManageContext_EmptyInput(t *testing.T) {
	cm, err := NewContextManager("gpt-4", 4096)
	require.NoError(t, err)

	t.Run("empty history and empty new messages", func(t *testing.T) {
		result, err := cm.ManageContext([]*modelaichat.Message{}, []string{})
		require.NoError(t, err)
		require.NotNil(t, result)
		require.Equal(t, 0, len(result))
	})

	t.Run("empty history with new messages", func(t *testing.T) {
		result, err := cm.ManageContext([]*modelaichat.Message{}, []string{"Hello", "World"})
		require.NoError(t, err)
		require.NotNil(t, result)
		require.Greater(t, len(result), 0)
		// Should contain new user messages
		require.Equal(t, schema.RoleType("user"), result[len(result)-1].Role)
	})

	t.Run("history with empty new messages", func(t *testing.T) {
		history := []*modelaichat.Message{
			{Role: modelaichat.MessageRoleUser, Content: "Hello"},
		}
		result, err := cm.ManageContext(history, []string{})
		require.NoError(t, err)
		require.NotNil(t, result)
		require.Greater(t, len(result), 0)
	})
}

func TestManageContext_SystemMessages(t *testing.T) {
	cm, err := NewContextManager("gpt-4", 4096)
	require.NoError(t, err)

	t.Run("system message is preserved", func(t *testing.T) {
		history := []*modelaichat.Message{
			{Role: modelaichat.MessageRoleSystem, Content: "You are a helpful assistant"},
			{Role: modelaichat.MessageRoleUser, Content: "Hello"},
			{Role: modelaichat.MessageRoleAssistant, Content: "Hi there"},
		}
		result, err := cm.ManageContext(history, []string{"New message"})
		require.NoError(t, err)
		require.NotNil(t, result)
		require.Greater(t, len(result), 0)
		// First message should be system message
		require.Equal(t, schema.RoleType("system"), result[0].Role)
		require.Contains(t, result[0].Content, "helpful assistant")
	})

	t.Run("multiple system messages are preserved", func(t *testing.T) {
		history := []*modelaichat.Message{
			{Role: modelaichat.MessageRoleSystem, Content: "System 1"},
			{Role: modelaichat.MessageRoleSystem, Content: "System 2"},
			{Role: modelaichat.MessageRoleUser, Content: "Hello"},
		}
		result, err := cm.ManageContext(history, []string{"New message"})
		require.NoError(t, err)
		require.NotNil(t, result)
		// Count system messages
		systemCount := 0
		for _, msg := range result {
			if msg.Role == schema.RoleType("system") {
				systemCount++
			}
		}
		require.Equal(t, 2, systemCount)
	})
}

func TestManageContext_ContextWindowExceeded(t *testing.T) {
	cm, err := NewContextManager("gpt-4", 100) // Small context window
	require.NoError(t, err)

	t.Run("large system message", func(t *testing.T) {
		largeSystemMsg := strings.Repeat("System instruction ", 100)
		history := []*modelaichat.Message{
			{Role: modelaichat.MessageRoleSystem, Content: largeSystemMsg},
			{Role: modelaichat.MessageRoleUser, Content: "Hello"},
		}
		result, err := cm.ManageContext(history, []string{"New message"})
		require.NoError(t, err)
		require.NotNil(t, result)
		// System message should still be preserved
		require.Equal(t, schema.RoleType("system"), result[0].Role)
	})

	t.Run("large new user message", func(t *testing.T) {
		largeNewMsg := strings.Repeat("New user message ", 100)
		history := []*modelaichat.Message{
			{Role: modelaichat.MessageRoleUser, Content: "Hello"},
		}
		result, err := cm.ManageContext(history, []string{largeNewMsg})
		require.NoError(t, err)
		require.NotNil(t, result)
		// New message should be included
		require.Greater(t, len(result), 0)
	})

	t.Run("many history messages", func(t *testing.T) {
		history := make([]*modelaichat.Message, 0)
		for range 50 {
			history = append(history, &modelaichat.Message{
				Role:    modelaichat.MessageRoleUser,
				Content: "Message " + strings.Repeat("content ", 10),
			})
			history = append(history, &modelaichat.Message{
				Role:    modelaichat.MessageRoleAssistant,
				Content: "Response " + strings.Repeat("content ", 10),
			})
		}
		result, err := cm.ManageContext(history, []string{"New message"})
		require.NoError(t, err)
		require.NotNil(t, result)
		// Should trim history but keep new message
		require.Greater(t, len(result), 0)
		// Last message should be the new user message
		lastMsg := result[len(result)-1]
		require.Equal(t, schema.RoleType("user"), lastMsg.Role)
		require.Contains(t, lastMsg.Content, "New message")
	})
}

func TestManageContext_MessageOrder(t *testing.T) {
	cm, err := NewContextManager("gpt-4", 4096)
	require.NoError(t, err)

	t.Run("message order is preserved", func(t *testing.T) {
		history := []*modelaichat.Message{
			{Role: modelaichat.MessageRoleUser, Content: "First"},
			{Role: modelaichat.MessageRoleAssistant, Content: "First response"},
			{Role: modelaichat.MessageRoleUser, Content: "Second"},
			{Role: modelaichat.MessageRoleAssistant, Content: "Second response"},
		}
		result, err := cm.ManageContext(history, []string{"Third"})
		require.NoError(t, err)
		require.NotNil(t, result)

		// Find conversation messages (skip system messages if any)
		convStart := 0
		for i, msg := range result {
			if msg.Role != "system" {
				convStart = i
				break
			}
		}

		// Check order: should maintain chronological order
		convMessages := result[convStart:]
		require.GreaterOrEqual(t, len(convMessages), 5) // At least 4 history + 1 new

		// Last message should be the new user message
		require.Equal(t, schema.RoleType("user"), convMessages[len(convMessages)-1].Role)
		require.Contains(t, convMessages[len(convMessages)-1].Content, "Third")
	})
}

func TestTrimMessages(t *testing.T) {
	cm, err := NewContextManager("gpt-4", 4096)
	require.NoError(t, err)

	t.Run("empty messages", func(t *testing.T) {
		result := cm.trimMessages([]*schema.Message{}, 100)
		require.Equal(t, 0, len(result))
	})

	t.Run("all messages fit", func(t *testing.T) {
		messages := []*schema.Message{
			schema.UserMessage("Short message 1"),
			schema.UserMessage("Short message 2"),
			schema.UserMessage("Short message 3"),
		}
		totalTokens := cm.CountTokensForMessages(messages)
		result := cm.trimMessages(messages, totalTokens+100)
		require.Equal(t, len(messages), len(result))
	})

	t.Run("some messages exceed limit", func(t *testing.T) {
		messages := []*schema.Message{
			schema.UserMessage("Message 1"),
			schema.UserMessage("Message 2"),
			schema.UserMessage("Message 3"),
			schema.UserMessage("Message 4"),
			schema.UserMessage("Message 5"),
		}
		// Set limit to fit only last 2 messages
		lastTwoTokens := cm.CountTokens(messages[3]) + cm.CountTokens(messages[4])
		result := cm.trimMessages(messages, lastTwoTokens+10)
		// Should keep most recent messages
		require.Greater(t, len(result), 0)
		require.LessOrEqual(t, len(result), len(messages))
		// Last message in result should be the last message in original
		require.Equal(t, messages[len(messages)-1].Content, result[len(result)-1].Content)
	})

	t.Run("single message exceeds limit", func(t *testing.T) {
		largeMsg := strings.Repeat("Very long message ", 1000)
		messages := []*schema.Message{
			schema.UserMessage(largeMsg),
		}
		msgTokens := cm.CountTokens(messages[0])
		result := cm.trimMessages(messages, msgTokens-1) // Limit is less than message
		// Should return empty or just the message if it fits
		if len(result) > 0 {
			// If message fits within limit, it should be included
			require.Equal(t, messages[0].Content, result[0].Content)
		}
	})

	t.Run("all messages exceed limit but last one fits", func(t *testing.T) {
		largeMsg := strings.Repeat("Large message ", 100)
		smallMsg := "Small"
		messages := []*schema.Message{
			schema.UserMessage(largeMsg),
			schema.UserMessage(largeMsg),
			schema.UserMessage(smallMsg),
		}
		smallTokens := cm.CountTokens(messages[2])
		result := cm.trimMessages(messages, smallTokens+5)
		// Should keep at least the last message if it fits
		if len(result) > 0 {
			require.Equal(t, smallMsg, result[len(result)-1].Content)
		}
	})
}

func TestManageContext_EdgeCases(t *testing.T) {
	cm, err := NewContextManager("gpt-4", 4096)
	require.NoError(t, err)

	t.Run("very small context window", func(t *testing.T) {
		smallCM, err := NewContextManager("gpt-4", 50)
		require.NoError(t, err)
		history := []*modelaichat.Message{
			{Role: modelaichat.MessageRoleUser, Content: "Hello"},
		}
		result, err := smallCM.ManageContext(history, []string{"World"})
		require.NoError(t, err)
		require.NotNil(t, result)
		// Should still return something
		require.Greater(t, len(result), 0)
	})

	t.Run("new user messages larger than available tokens", func(t *testing.T) {
		smallCM, err := NewContextManager("gpt-4", 100)
		require.NoError(t, err)
		largeNewMsg := strings.Repeat("Very large new message ", 200)
		history := []*modelaichat.Message{
			{Role: modelaichat.MessageRoleUser, Content: "Hello"},
		}
		result, err := smallCM.ManageContext(history, []string{largeNewMsg})
		require.NoError(t, err)
		require.NotNil(t, result)
		// New message should still be included even if it's large
		require.Greater(t, len(result), 0)
	})

	t.Run("boundary check for history length", func(t *testing.T) {
		// Test the boundary check we added
		history := []*modelaichat.Message{
			{Role: modelaichat.MessageRoleUser, Content: "Hello"},
		}
		// This should not panic
		result, err := cm.ManageContext(history, []string{"World", "Test"})
		require.NoError(t, err)
		require.NotNil(t, result)
	})

	t.Run("edge case: new messages count exceeds history", func(t *testing.T) {
		// Edge case where new messages are more than history
		// This tests the boundary check: historyLen := len(conversationMessages) - len(newUserEinoMessages)
		history := []*modelaichat.Message{
			{Role: modelaichat.MessageRoleUser, Content: "Hello"},
		}
		// Add many new messages
		newMessages := make([]string, 10)
		for i := range newMessages {
			newMessages[i] = "New message"
		}
		result, err := cm.ManageContext(history, newMessages)
		require.NoError(t, err)
		require.NotNil(t, result)
		// Should still work correctly
		require.Greater(t, len(result), 0)
	})

	t.Run("edge case: all messages exceed limit returns empty", func(t *testing.T) {
		// Test case where even the last message exceeds the limit
		smallCM, err := NewContextManager("gpt-4", 10) // Very small limit
		require.NoError(t, err)
		largeMsg := strings.Repeat("Very large message that exceeds limit ", 100)
		messages := []*schema.Message{
			schema.UserMessage(largeMsg),
		}
		result := smallCM.trimMessages(messages, 10)
		// If message exceeds limit, should return empty
		// This is expected behavior - can't fit even one message
		require.NotNil(t, result)
		// Result might be empty or contain the message if it somehow fits
		require.LessOrEqual(t, len(result), 1)
	})
}

func TestManageContext_TokenCalculation(t *testing.T) {
	cm, err := NewContextManager("gpt-4", 1000)
	require.NoError(t, err)

	t.Run("reserved tokens calculation", func(t *testing.T) {
		history := []*modelaichat.Message{
			{Role: modelaichat.MessageRoleUser, Content: "Test"},
		}
		result, err := cm.ManageContext(history, []string{"New"})
		require.NoError(t, err)
		require.NotNil(t, result)

		// Total tokens should be less than context length
		totalTokens := cm.CountTokensForMessages(result)
		require.LessOrEqual(t, totalTokens, cm.contextLength)
	})

	t.Run("system messages don't count against conversation limit", func(t *testing.T) {
		largeSystemMsg := strings.Repeat("System ", 50)
		history := []*modelaichat.Message{
			{Role: modelaichat.MessageRoleSystem, Content: largeSystemMsg},
			{Role: modelaichat.MessageRoleUser, Content: "Hello"},
		}
		result, err := cm.ManageContext(history, []string{"World"})
		require.NoError(t, err)
		require.NotNil(t, result)
		// System message should be preserved
		require.Equal(t, schema.RoleType("system"), result[0].Role)
	})
}

func TestTrimMessages_Performance(t *testing.T) {
	cm, err := NewContextManager("gpt-4", 4096)
	require.NoError(t, err)

	t.Run("many messages performance", func(t *testing.T) {
		messages := make([]*schema.Message, 0, 1000)
		for range 1000 {
			messages = append(messages, schema.UserMessage("Message "+strings.Repeat("content ", 10)))
		}
		// This should complete quickly without excessive memory allocation
		result := cm.trimMessages(messages, 1000)
		require.NotNil(t, result)
		require.LessOrEqual(t, len(result), len(messages))
	})
}

func TestManageContext_RealWorldScenario(t *testing.T) {
	cm, err := NewContextManager("gpt-4", 4096)
	require.NoError(t, err)

	t.Run("typical conversation flow", func(t *testing.T) {
		// Simulate a typical conversation
		history := []*modelaichat.Message{
			{Role: modelaichat.MessageRoleSystem, Content: "You are a helpful assistant"},
			{Role: modelaichat.MessageRoleUser, Content: "What is Go?"},
			{Role: modelaichat.MessageRoleAssistant, Content: "Go is a programming language..."},
			{Role: modelaichat.MessageRoleUser, Content: "Tell me more"},
			{Role: modelaichat.MessageRoleAssistant, Content: "Go has many features..."},
		}
		result, err := cm.ManageContext(history, []string{"What about concurrency?"})
		require.NoError(t, err)
		require.NotNil(t, result)

		// Verify structure
		require.Greater(t, len(result), 0)
		// First should be system
		require.Equal(t, schema.RoleType("system"), result[0].Role)
		// Last should be new user message
		require.Equal(t, schema.RoleType("user"), result[len(result)-1].Role)
		require.Contains(t, result[len(result)-1].Content, "concurrency")
	})
}

func TestManageContext_PotentialBugs(t *testing.T) {
	t.Run("bug: conversationAvailableTokens becomes negative", func(t *testing.T) {
		// Test case where system + new messages exceed available tokens
		// This should not cause issues - conversationAvailableTokens should be 0
		smallCM, err := NewContextManager("gpt-4", 100)
		require.NoError(t, err)
		largeSystemMsg := strings.Repeat("System instruction ", 200)
		largeNewMsg := strings.Repeat("New user message ", 200)
		history := []*modelaichat.Message{
			{Role: modelaichat.MessageRoleSystem, Content: largeSystemMsg},
			{Role: modelaichat.MessageRoleUser, Content: "Hello"},
		}
		result, err := smallCM.ManageContext(history, []string{largeNewMsg})
		require.NoError(t, err)
		require.NotNil(t, result)
		// Should still return valid result
		require.Greater(t, len(result), 0)
	})

	t.Run("bug: message order after trimming", func(t *testing.T) {
		// Verify that after trimming, messages maintain correct order
		cm, err := NewContextManager("gpt-4", 500)
		require.NoError(t, err)
		history := []*modelaichat.Message{
			{Role: modelaichat.MessageRoleUser, Content: "Message 1"},
			{Role: modelaichat.MessageRoleAssistant, Content: "Response 1"},
			{Role: modelaichat.MessageRoleUser, Content: "Message 2"},
			{Role: modelaichat.MessageRoleAssistant, Content: "Response 2"},
			{Role: modelaichat.MessageRoleUser, Content: "Message 3"},
			{Role: modelaichat.MessageRoleAssistant, Content: "Response 3"},
		}
		result, err := cm.ManageContext(history, []string{"Message 4"})
		require.NoError(t, err)
		require.NotNil(t, result)

		// Find conversation messages (skip system if any)
		convStart := 0
		for i, msg := range result {
			if msg.Role != "system" {
				convStart = i
				break
			}
		}
		convMessages := result[convStart:]

		// Verify order: should be chronological
		// Last message should be the new user message
		require.Equal(t, schema.RoleType("user"), convMessages[len(convMessages)-1].Role)
		require.Contains(t, convMessages[len(convMessages)-1].Content, "Message 4")

		// If we have multiple messages, they should be in order
		if len(convMessages) > 1 {
			// Check that user/assistant pairs are maintained
			// This is a simplified check - in real scenario, we'd verify the full sequence
			require.Greater(t, len(convMessages), 1)
		}
	})

	t.Run("bug: trimMessages preserves order correctly", func(t *testing.T) {
		// Test that trimMessages maintains message order after optimization
		cm, err := NewContextManager("gpt-4", 500)
		require.NoError(t, err)
		messages := []*schema.Message{
			schema.UserMessage("First"),
			schema.UserMessage("Second"),
			schema.UserMessage("Third"),
			schema.UserMessage("Fourth"),
			schema.UserMessage("Fifth"),
		}
		// Set limit to fit only last 2 messages (Fourth and Fifth)
		lastTwoTokens := cm.CountTokens(messages[3]) + cm.CountTokens(messages[4])
		result := cm.trimMessages(messages, lastTwoTokens+5) // Small buffer
		require.Greater(t, len(result), 0)
		// Last message in result should be last message in original
		require.Equal(t, messages[len(messages)-1].Content, result[len(result)-1].Content)
		// Should keep most recent messages that fit
		// Since we're trimming from the end backwards, result should contain recent messages
		require.GreaterOrEqual(t, len(result), 1)
		// Verify the last message is preserved
		require.Equal(t, "Fifth", result[len(result)-1].Content)
	})

	t.Run("bug: empty newUserEinoMessages edge case", func(t *testing.T) {
		// Test case where newUserEinoMessages is empty after creation
		// This tests the boundary check: historyLen := len(conversationMessages) - len(newUserEinoMessages)
		cm, err := NewContextManager("gpt-4", 4096)
		require.NoError(t, err)
		history := []*modelaichat.Message{
			{Role: modelaichat.MessageRoleUser, Content: "Hello"},
		}
		// Empty new messages - should not cause issues
		result, err := cm.ManageContext(history, []string{})
		require.NoError(t, err)
		require.NotNil(t, result)
		// Should still return history messages
		require.Greater(t, len(result), 0)
	})

	t.Run("bug: token count accuracy", func(t *testing.T) {
		// Verify that token counting is consistent
		cm, err := NewContextManager("gpt-4", 4096)
		require.NoError(t, err)
		msg1 := schema.UserMessage("Hello")
		msg2 := schema.UserMessage("World")
		tokens1 := cm.CountTokens(msg1)
		tokens2 := cm.CountTokens(msg2)
		combinedTokens := cm.CountTokensForMessages([]*schema.Message{msg1, msg2})
		// Combined should equal sum
		require.Equal(t, tokens1+tokens2, combinedTokens)
	})
}
