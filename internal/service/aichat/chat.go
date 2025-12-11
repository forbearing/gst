package serviceaichat

import (
	"io"
	"time"

	"github.com/cloudwego/eino-ext/components/model/claude"
	"github.com/cloudwego/eino-ext/components/model/ollama"
	"github.com/cloudwego/eino-ext/components/model/openai"
	einomodel "github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/schema"
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
	messages := make([]*modelaichat.Message, 0)
	if err := database.Database[*modelaichat.Message](ctx.DatabaseContext()).
		WithQuery(&modelaichat.Message{ConversationID: conversation.ID}).
		List(&messages); err != nil {
		return nil, errors.Wrap(err, "failed to get message history")
	}

	// 5. Build eino messages and save new user messages to database
	einoMessages := make([]*schema.Message, 0)

	// Add history messages to eino messages
	for _, msg := range messages {
		switch msg.Role {
		case modelaichat.MessageRoleSystem:
			einoMessages = append(einoMessages, schema.SystemMessage(msg.Content))
		case modelaichat.MessageRoleUser:
			einoMessages = append(einoMessages, schema.UserMessage(msg.Content))
		case modelaichat.MessageRoleAssistant:
			einoMessages = append(einoMessages, schema.AssistantMessage(msg.Content, nil))
		}
	}

	// Add new user messages to eino messages and save to database
	for _, content := range req.Messages {
		// Add to eino messages first
		einoMessages = append(einoMessages, schema.UserMessage(content))

		// Save to database
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

	// 6. Create AI model client based on provider type
	config := provider.Config.Data()
	var chatModel einomodel.ToolCallingChatModel
	var err error

	switch provider.Type {
	case modelaichat.ProviderAnthropic:
		if chatModel, err = claude.NewChatModel(ctx.Context(), &claude.Config{
			APIKey:    config.APIKey,
			Model:     aiModel.ModelID,
			MaxTokens: aiModel.Config.Data().MaxTokens,
			BaseURL:   &config.BaseURL,
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

	// 7. Create assistant message in database
	assistantMsg := &modelaichat.Message{
		ConversationID: conversation.ID,
		ModelID:        req.ModelID,
		Role:           modelaichat.MessageRoleAssistant,
		Status:         modelaichat.MessageStatusPending,
	}

	if err := database.Database[*modelaichat.Message](ctx.DatabaseContext()).Create(assistantMsg); err != nil {
		return nil, errors.Wrap(err, "failed to create assistant message")
	}

	// 8. Handle streaming or non-streaming response
	if req.Stream {
		return s.handleStreaming(ctx, chatModel, einoMessages, assistantMsg, conversation)
	}
	return s.handleNonStreaming(ctx, chatModel, einoMessages, assistantMsg, conversation)
}

// handleStreaming handles streaming response
func (s *ChatCompletion) handleStreaming(
	ctx *types.ServiceContext,
	chatModel einomodel.ToolCallingChatModel,
	einoMessages []*schema.Message,
	assistantMsg *modelaichat.Message,
	conversation *modelaichat.Conversation,
) (*modelaichat.ChatCompletionRsp, error) {
	_ = conversation
	// Start streaming
	stream, err := chatModel.Stream(ctx.Context(), einoMessages)
	if err != nil {
		assistantMsg.Status = modelaichat.MessageStatusFailed
		assistantMsg.ErrMessage = err.Error()
		assistantMsg.StopReason = util.ValueOf(modelaichat.StopReasonError)
		if e := database.Database[*modelaichat.Message](ctx.DatabaseContext()).Update(assistantMsg); e != nil {
			s.Errorw("failed to update message", "error", e)
		}
		return nil, errors.Wrap(err, "failed to start streaming")
	}
	defer stream.Close()

	// Update message status
	assistantMsg.Status = modelaichat.MessageStatusStreaming
	msgDB := database.Database[*modelaichat.Message](ctx.DatabaseContext())
	if err := msgDB.Update(assistantMsg); err != nil {
		s.Errorw("failed to update message", "error", err)
	}

	var fullContent string
	startTime := time.Now()

	// Stream response using SSE
	defer func() {
		_ = ctx.SSE().Done()
	}()

	ctx.SSE().Stream(func(w io.Writer) bool {
		chunk, err := stream.Recv()
		if errors.Is(err, io.EOF) {
			// Stream ended, try to extract usage information if available
			assistantMsg.Status = modelaichat.MessageStatusCompleted
			assistantMsg.Content = fullContent
			assistantMsg.StopReason = util.ValueOf(modelaichat.StopReasonEndTurn)
			assistantMsg.LatencyMs = time.Since(startTime).Milliseconds()

			if chunk.ResponseMeta != nil || chunk.ResponseMeta.Usage != nil {
				assistantMsg.PromptTokens = chunk.ResponseMeta.Usage.PromptTokens
				assistantMsg.CompletionTokens = chunk.ResponseMeta.Usage.CompletionTokens
				assistantMsg.TotalTokens = chunk.ResponseMeta.Usage.TotalTokens
			}

			if err = msgDB.Update(assistantMsg); err != nil {
				s.Errorw("failed to update message", "error", err)
			}
			return false
		}

		if err != nil {
			// Error occurred
			assistantMsg.Status = modelaichat.MessageStatusFailed
			assistantMsg.ErrMessage = err.Error()
			assistantMsg.StopReason = util.ValueOf(modelaichat.StopReasonError)
			assistantMsg.LatencyMs = time.Since(startTime).Milliseconds()
			if err = msgDB.Update(assistantMsg); err != nil {
				s.Errorw("failed to update message", "error", err)
			}

			_ = ctx.Encode(w, types.Event{
				Event: "error",
				Data: map[string]any{
					"error": err.Error(),
				},
			})
			return false
		}

		// Append chunk content
		fullContent += chunk.Content

		// Send chunk via SSE
		_ = ctx.Encode(w, types.Event{
			Event: "message",
			Data: map[string]any{
				"content": chunk.Content,
				"delta":   chunk.Content, // For compatibility
			},
		})

		return true
	})

	return nil, nil
}

// handleNonStreaming handles non-streaming response
func (s *ChatCompletion) handleNonStreaming(
	ctx *types.ServiceContext,
	chatModel einomodel.ToolCallingChatModel,
	einoMessages []*schema.Message,
	assistantMsg *modelaichat.Message,
	conversation *modelaichat.Conversation,
) (*modelaichat.ChatCompletionRsp, error) {
	_ = conversation
	startTime := time.Now()

	// Generate response
	response, err := chatModel.Generate(ctx.Context(), einoMessages)
	if err != nil {
		assistantMsg.Status = modelaichat.MessageStatusFailed
		assistantMsg.ErrMessage = err.Error()
		assistantMsg.StopReason = util.ValueOf(modelaichat.StopReasonError)
		if e := database.Database[*modelaichat.Message](ctx.DatabaseContext()).Update(assistantMsg); e != nil {
			err = errors.Join(err, errors.Wrap(e, "failed to update message"))
		}
		return nil, errors.Wrap(err, "failed to generate response")
	}

	assistantMsg.Status = modelaichat.MessageStatusCompleted
	assistantMsg.Content = response.Content
	assistantMsg.StopReason = util.ValueOf(modelaichat.StopReasonEndTurn)
	assistantMsg.LatencyMs = time.Since(startTime).Milliseconds()

	// Extract token usage from response
	if response.ResponseMeta != nil || response.ResponseMeta.Usage != nil {
		assistantMsg.PromptTokens = response.ResponseMeta.Usage.PromptTokens
		assistantMsg.CompletionTokens = response.ResponseMeta.Usage.CompletionTokens
		assistantMsg.TotalTokens = response.ResponseMeta.Usage.TotalTokens
	}

	if err := database.Database[*modelaichat.Message](ctx.DatabaseContext()).Update(assistantMsg); err != nil {
		return nil, errors.Wrap(err, "failed to update message")
	}

	return &modelaichat.ChatCompletionRsp{
		ConversationID: conversation.ID,
		MessageID:      assistantMsg.ID,
		Content:        response.Content,
	}, nil
}
