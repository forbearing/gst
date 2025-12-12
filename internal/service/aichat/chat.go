package serviceaichat

import (
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

type ChatCompletion struct {
	service.Base[*model.Empty, *modelaichat.ChatCompletionReq, *modelaichat.ChatCompletionRsp]
}

func (s *ChatCompletion) Create(ctx *types.ServiceContext, req *modelaichat.ChatCompletionReq) (*modelaichat.ChatCompletionRsp, error) {
	log := s.WithServiceContext(ctx, ctx.GetPhase())
	log.Infow("chat completion", "conversation_id", req.ConversationID, "model_id", req.ModelID, "stream", req.Stream)

	// Validate request
	if len(req.ModelID) == 0 {
		return nil, errors.New("model_id is required")
	}

	// 1. Get model information by model_id
	aiModel := new(modelaichat.Model)
	if err := database.Database[*modelaichat.Model](ctx.DatabaseContext()).
		WithQuery(&modelaichat.Model{Base: model.Base{ID: req.ModelID}}).
		First(aiModel); err != nil {
		return nil, errors.Wrapf(err, "failed to get ai model: %s", req.ModelID)
	}

	// 2. Get provider information by provider_id from model
	provider := new(modelaichat.Provider)
	if err := database.Database[*modelaichat.Provider](ctx.DatabaseContext()).
		WithQuery(&modelaichat.Provider{Base: model.Base{ID: aiModel.ProviderID}}).
		First(provider); err != nil {
		return nil, errors.Wrapf(err, "failed to get provider: %s", aiModel.ProviderID)
	}

	// 3. Get or create conversation
	var conversation *modelaichat.Conversation
	if len(req.ConversationID) > 0 {
		conversation = new(modelaichat.Conversation)
		if err := database.Database[*modelaichat.Conversation](ctx.DatabaseContext()).
			WithQuery(&modelaichat.Conversation{Base: model.Base{ID: req.ConversationID}}).
			First(conversation); err != nil {
			return nil, errors.Wrapf(err, "failed to get conversation: %s", req.ConversationID)
		}
		// Verify conversation belongs to current user
		if conversation.UserID != ctx.UserID {
			return nil, errors.New("conversation does not belong to current user")
		}
	} else {
		// Create new conversation
		conversation = &modelaichat.Conversation{
			UserID:  ctx.UserID,
			Title:   "New Conversation",
			ModelID: req.ModelID,
		}
		if err := database.Database[*modelaichat.Conversation](ctx.DatabaseContext()).Create(conversation); err != nil {
			return nil, errors.Wrap(err, "failed to create conversation")
		}
	}

	// 4. Get conversation history messages
	// Filter out failed and inactive messages to ensure only valid messages are included
	messages := make([]*modelaichat.Message, 0)
	query := &modelaichat.Message{
		ConversationID: conversation.ID,
		Status:         modelaichat.MessageStatusCompleted,
		IsActive:       util.ValueOf(true), // Only include active messages
	}
	// Filter out failed and inactive messages
	// Only include completed messages that are active
	if err := database.Database[*modelaichat.Message](ctx.DatabaseContext()).
		WithQuery(query).
		WithOrder("created_at ASC").
		List(&messages); err != nil {
		return nil, errors.Wrap(err, "failed to get message history")
	}

	// 5. Save new user messages to database
	for _, content := range req.Messages {
		if err := database.Database[*modelaichat.Message](ctx.DatabaseContext()).Create(&modelaichat.Message{
			ConversationID: conversation.ID,
			ModelID:        req.ModelID,
			Role:           modelaichat.MessageRoleUser,
			Content:        content,
			Status:         modelaichat.MessageStatusCompleted,
		}); err != nil {
			return nil, errors.Wrap(err, "failed to create message")
		}
	}

	// 6. Manage context window using ContextManager
	// messages contains all historical messages (user + assistant) from database
	// req.Messages contains new user messages from current request
	// ContextManager will merge them and trim using sliding window strategy to fit within context length
	modelConfig := aiModel.Config.Data()
	contextLength := modelConfig.ContextLength
	if contextLength <= 0 {
		// Default context length if not specified
		contextLength = 4096
	}

	contextManager, err := NewContextManager(aiModel.ModelID, contextLength)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create context manager")
	}

	// ManageContext merges history messages and new messages, then trims to fit context window
	// It preserves system messages and uses sliding window for conversation history
	einoMessages, err := contextManager.ManageContext(messages, req.Messages)
	if err != nil {
		return nil, errors.Wrap(err, "failed to manage context")
	}

	// 7. Create AI model client based on provider type
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
			maxTokens = 4096 // Default max tokens for Anthropic
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
		// eino will auto set openai BaseURL.
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

	// 8. Create assistant message in database
	assistantMsg := &modelaichat.Message{
		ConversationID: conversation.ID,
		ModelID:        req.ModelID,
		Role:           modelaichat.MessageRoleAssistant,
		Status:         modelaichat.MessageStatusPending,
	}

	if err := database.Database[*modelaichat.Message](ctx.DatabaseContext()).Create(assistantMsg); err != nil {
		return nil, errors.Wrap(err, "failed to create assistant message")
	}

	// 9. Handle streaming or non-streaming response
	if req.Stream {
		return handleStreaming(ctx, log, chatModel, einoMessages, assistantMsg, conversation)
	}
	return handleNonStreaming(ctx, log, chatModel, einoMessages, assistantMsg, conversation)
}
