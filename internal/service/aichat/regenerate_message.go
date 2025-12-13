package serviceaichat

import (
	"time"

	"github.com/cloudwego/eino-ext/components/model/claude"
	"github.com/cloudwego/eino-ext/components/model/ollama"
	"github.com/cloudwego/eino-ext/components/model/openai"
	einomodel "github.com/cloudwego/eino/components/model"
	"github.com/cockroachdb/errors"
	"github.com/forbearing/gst/database"
	modelaichat "github.com/forbearing/gst/internal/model/aichat"
	"github.com/forbearing/gst/model"
	"github.com/forbearing/gst/service"
	"github.com/forbearing/gst/types"
	"github.com/forbearing/gst/util"
)

// RegenerateMessage handles regenerating an assistant message
type RegenerateMessage struct {
	service.Base[*model.Empty, *modelaichat.RegenerateMessageReq, *modelaichat.RegenerateMessageRsp]
}

func (s *RegenerateMessage) Create(ctx *types.ServiceContext, req *modelaichat.RegenerateMessageReq) (*modelaichat.RegenerateMessageRsp, error) {
	log := s.WithServiceContext(ctx, ctx.GetPhase())
	log.Infow("regenerate message", "message_id", req.MessageID, "stream", req.Stream)

	// Validate request
	if len(req.MessageID) == 0 {
		return nil, errors.New("message_id is required")
	}

	// 1. Get the assistant message to regenerate
	assistantMsg := new(modelaichat.Message)
	if err := database.Database[*modelaichat.Message](ctx.DatabaseContext()).Get(assistantMsg, req.MessageID); err != nil {
		return nil, errors.Wrapf(err, "failed to get message: %s", req.MessageID)
	}
	// Verify it's an assistant message
	if assistantMsg.Role != modelaichat.MessageRoleAssistant {
		return nil, errors.New("can only regenerate assistant messages")
	}

	// 2. Get conversation and verify ownership
	conversation := new(modelaichat.Conversation)
	if err := database.Database[*modelaichat.Conversation](ctx.DatabaseContext()).
		Get(conversation, assistantMsg.ConversationID); err != nil {
		return nil, errors.Wrapf(err, "failed to get conversation: %s", assistantMsg.ConversationID)
	}
	if conversation.UserID != ctx.UserID {
		return nil, errors.New("message does not belong to current user")
	}

	// 3. Find the user message that triggered this assistant message
	// Get the most recent user message before this assistant message
	if assistantMsg.CreatedAt == nil {
		return nil, errors.New("assistant message created_at is nil")
	}
	// Query for the most recent user message before the assistant message
	userMsg := new(modelaichat.Message)
	if err := database.Database[*modelaichat.Message](ctx.DatabaseContext()).
		WithQuery(&modelaichat.Message{
			ConversationID: conversation.ID,
			Role:           modelaichat.MessageRoleUser,
		}).
		WithTimeRange("created_at", time.Time{}, *assistantMsg.CreatedAt).
		WithOrder("created_at DESC").
		First(userMsg); err != nil {
		return nil, errors.Wrap(err, "failed to get user message before assistant message")
	}
	if userMsg.CreatedAt == nil {
		return nil, errors.New("user message created_at is nil")
	}

	// 4. Mark the original assistant message as inactive
	assistantMsg.IsActive = util.ValueOf(false)
	if err := database.Database[*modelaichat.Message](ctx.DatabaseContext()).Update(assistantMsg); err != nil {
		return nil, errors.Wrap(err, "failed to deactivate original message")
	}

	// 5. Get model and provider information using Get method
	aiModel := new(modelaichat.Model)
	if err := database.Database[*modelaichat.Model](ctx.DatabaseContext()).Get(aiModel, assistantMsg.ModelID); err != nil {
		return nil, errors.Wrapf(err, "failed to get ai model: %s", assistantMsg.ModelID)
	}

	provider := new(modelaichat.Provider)
	if err := database.Database[*modelaichat.Provider](ctx.DatabaseContext()).Get(provider, aiModel.ProviderID); err != nil {
		return nil, errors.Wrapf(err, "failed to get provider: %s", aiModel.ProviderID)
	}

	// 6. Get conversation history up to (and including) the user message using WithTimeRange
	// Exclude the original assistant message and any messages after it
	historyMessages := make([]*modelaichat.Message, 0)
	if err := database.Database[*modelaichat.Message](ctx.DatabaseContext()).
		WithQuery(&modelaichat.Message{
			ConversationID: conversation.ID,
			Status:         modelaichat.MessageStatusCompleted,
			IsActive:       util.ValueOf(true),
		}).
		WithTimeRange("created_at", time.Time{}, *userMsg.CreatedAt).
		WithOrder("created_at ASC").
		List(&historyMessages); err != nil {
		return nil, errors.Wrap(err, "failed to get conversation history")
	}

	// 7. Create new assistant message with ParentID pointing to original message
	newAssistantMsg := &modelaichat.Message{
		ConversationID:  conversation.ID,
		ModelID:         assistantMsg.ModelID,
		Role:            modelaichat.MessageRoleAssistant,
		Status:          modelaichat.MessageStatusPending,
		ParentID:        util.ValueOf(assistantMsg.ID),
		RegenerateCount: assistantMsg.RegenerateCount + 1,
		IsActive:        util.ValueOf(true),
	}
	if err := database.Database[*modelaichat.Message](ctx.DatabaseContext()).Create(newAssistantMsg); err != nil {
		return nil, errors.Wrap(err, "failed to create new assistant message")
	}

	// 8. Manage context window
	modelConfig := aiModel.Config.Data()
	contextLength := modelConfig.ContextLength
	if contextLength <= 0 {
		contextLength = 4096
	}
	contextManager, err := NewContextManager(aiModel.ModelID, contextLength)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create context manager")
	}

	// Convert history messages to eino messages
	// The user message content is already in historyMessages, so we pass empty slice for new messages
	einoMessages, err := contextManager.ManageContext(historyMessages, []string{})
	if err != nil {
		return nil, errors.Wrap(err, "failed to manage context")
	}

	// 9. Create AI model client
	config := provider.Config.Data()
	var chatModel einomodel.ToolCallingChatModel
	switch provider.Type {
	case modelaichat.ProviderAnthropic:
		baseURL := config.BaseURL
		if baseURL == "" {
			baseURL = "https://api.anthropic.com"
		}
		maxTokens := aiModel.Config.Data().MaxTokens
		if maxTokens <= 0 {
			maxTokens = 4096
		}
		if chatModel, err = claude.NewChatModel(ctx.Context(), &claude.Config{
			APIKey:    config.APIKey,
			Model:     aiModel.ModelID,
			MaxTokens: maxTokens,
			BaseURL:   &baseURL,
		}); err != nil {
			return nil, errors.Wrap(err, "failed to create claude client")
		}
	case modelaichat.ProviderOpenAI:
		if chatModel, err = openai.NewChatModel(ctx.Context(), &openai.ChatModelConfig{
			APIKey:    config.APIKey,
			Model:     aiModel.ModelID,
			MaxTokens: util.ValueOf(aiModel.Config.Data().MaxTokens),
			BaseURL:   config.BaseURL,
		}); err != nil {
			return nil, errors.Wrap(err, "failed to create openai client")
		}
	case modelaichat.ProviderLocal:
		baseURL := config.BaseURL
		if baseURL == "" {
			baseURL = "http://localhost:11434"
		}
		if chatModel, err = ollama.NewChatModel(ctx.Context(), &ollama.ChatModelConfig{
			BaseURL: baseURL,
			Model:   aiModel.ModelID,
		}); err != nil {
			return nil, errors.Wrap(err, "failed to create ollama client")
		}
	default:
		return nil, errors.Newf("unsupported provider type: %s", provider.Type)
	}

	// 10. Handle streaming or non-streaming response using shared functions
	if req.Stream {
		// For streaming mode, handleStreaming returns nil, nil as response is sent via SSE
		if _, err = handleStreaming(ctx, log, chatModel, einoMessages, newAssistantMsg, conversation); err != nil {
			return nil, err
		}
		// Return response with message ID and conversation ID for streaming mode
		return &modelaichat.RegenerateMessageRsp{
			MessageID:      newAssistantMsg.ID,
			ConversationID: conversation.ID,
		}, nil
	}

	// For non-streaming mode, handleNonStreaming returns the full response
	chatRsp, err := handleNonStreaming(ctx, log, chatModel, einoMessages, newAssistantMsg, conversation)
	if err != nil {
		return nil, err
	}

	// Convert ChatCompletionRsp to RegenerateMessageRsp
	return &modelaichat.RegenerateMessageRsp{
		MessageID:      chatRsp.MessageID,
		ConversationID: chatRsp.ConversationID,
		Content:        chatRsp.Content,
	}, nil
}
