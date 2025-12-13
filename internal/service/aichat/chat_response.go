package serviceaichat

import (
	"context"
	"io"
	"time"

	einomodel "github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/schema"
	"github.com/cockroachdb/errors"
	"github.com/forbearing/gst/database"
	modelaichat "github.com/forbearing/gst/internal/model/aichat"
	"github.com/forbearing/gst/types"
	"github.com/forbearing/gst/util"
)

// handleStreaming handles streaming response
func handleStreaming(
	ctx *types.ServiceContext,
	log types.Logger,
	chatModel einomodel.ToolCallingChatModel,
	einoMessages []*schema.Message,
	assistantMsg *modelaichat.Message,
	conversation *modelaichat.Conversation,
) (*modelaichat.ChatCompletionRsp, error) {
	_ = conversation
	db := database.Database[*modelaichat.Message](ctx.DatabaseContext())

	// Create a cancellable context for this stream
	streamCtx, cancel := context.WithCancel(ctx.Context())
	defer cancel()

	// Register the stream with the manager
	streamManager := GetStreamManager()
	if err := streamManager.RegisterStream(assistantMsg.ID, cancel); err != nil {
		log.Warnw("failed to register stream", "error", err)
	}
	defer streamManager.UnregisterStream(assistantMsg.ID)

	// Start streaming
	stream, err := chatModel.Stream(streamCtx, einoMessages)
	if err != nil {
		assistantMsg.Status = modelaichat.MessageStatusFailed
		assistantMsg.ErrMessage = err.Error()
		assistantMsg.StopReason = util.ValueOf(modelaichat.StopReasonError)
		if e := db.Update(assistantMsg); e != nil {
			log.Errorw("failed to update message", "error", e)
		}
		return nil, errors.Wrap(err, "failed to start streaming")
	}
	defer stream.Close()

	// Update message status
	assistantMsg.Status = modelaichat.MessageStatusStreaming
	if err := db.Update(assistantMsg); err != nil {
		log.Errorw("failed to update message", "error", err)
	}

	var fullContent string
	startTime := time.Now()

	// Stream response using SSE
	defer func() {
		_ = ctx.SSE().Done()
	}()

	// Channel to receive stream chunks
	type streamResult struct {
		chunk *schema.Message
		err   error
	}
	chunkChan := make(chan streamResult, 1)

	ctx.SSE().Stream(func(w io.Writer) bool {
		// Start receiving chunk in a goroutine to avoid blocking
		go func() {
			chunk, err := stream.Recv()
			// Check if context was canceled before sending to channel to avoid blocking
			select {
			case <-streamCtx.Done():
				// Context canceled, don't send to channel
				return
			case chunkChan <- streamResult{chunk: chunk, err: err}:
			}
		}()

		// Wait for either context cancellation or stream data
		select {
		case <-streamCtx.Done():
			log.Infow("stream canceled", "error", streamCtx.Err())
			// Stream was canceled
			assistantMsg.Status = modelaichat.MessageStatusStopped
			assistantMsg.Content = fullContent
			assistantMsg.StopReason = util.ValueOf(modelaichat.StopReasonUser)
			assistantMsg.IsActive = util.ValueOf(false) // Mark as inactive when stopped
			assistantMsg.LatencyMs = time.Since(startTime).Milliseconds()

			if err := db.Update(assistantMsg); err != nil {
				log.Errorw("failed to update message", "error", err)
			}

			_ = ctx.Encode(w, types.Event{
				Event: "stopped",
				Data: map[string]any{
					"message_id": assistantMsg.ID,
				},
			})
			log.Info("stream stopped by user")
			return false

		case result := <-chunkChan:
			chunk, err := result.chunk, result.err

			if errors.Is(err, io.EOF) {
				// Stream ended, try to extract usage information if available
				assistantMsg.Status = modelaichat.MessageStatusCompleted
				assistantMsg.Content = fullContent
				assistantMsg.StopReason = util.ValueOf(modelaichat.StopReasonEndTurn)
				assistantMsg.IsActive = util.ValueOf(true) // Mark as active when completed
				assistantMsg.LatencyMs = time.Since(startTime).Milliseconds()

				if chunk != nil && chunk.ResponseMeta != nil && chunk.ResponseMeta.Usage != nil {
					assistantMsg.PromptTokens = chunk.ResponseMeta.Usage.PromptTokens
					assistantMsg.CompletionTokens = chunk.ResponseMeta.Usage.CompletionTokens
					assistantMsg.TotalTokens = chunk.ResponseMeta.Usage.TotalTokens
				}

				if err = db.Update(assistantMsg); err != nil {
					log.Errorw("failed to update message", "error", err)
				}
				log.Info("stream ended")
				return false
			}

			if err != nil {
				// Error occurred
				assistantMsg.Status = modelaichat.MessageStatusFailed
				assistantMsg.ErrMessage = err.Error()
				assistantMsg.StopReason = util.ValueOf(modelaichat.StopReasonError)
				assistantMsg.LatencyMs = time.Since(startTime).Milliseconds()
				if err = db.Update(assistantMsg); err != nil {
					log.Errorw("failed to update message", "error", err)
				}

				_ = ctx.Encode(w, types.Event{
					Event: "error",
					Data: map[string]any{
						"error": err.Error(),
					},
				})
				return false
			}

			if chunk == nil {
				log.Errorw("received nil chunk")
				return true // Continue to next iteration
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
		}
	})

	return nil, nil
}

// handleNonStreaming handles non-streaming response
func handleNonStreaming(
	ctx *types.ServiceContext,
	log types.Logger,
	chatModel einomodel.ToolCallingChatModel,
	einoMessages []*schema.Message,
	assistantMsg *modelaichat.Message,
	conversation *modelaichat.Conversation,
) (*modelaichat.ChatCompletionRsp, error) {
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
	assistantMsg.Content = util.Deref(response).Content
	assistantMsg.StopReason = util.ValueOf(modelaichat.StopReasonEndTurn)
	assistantMsg.IsActive = util.ValueOf(true) // Mark as active when completed
	assistantMsg.LatencyMs = time.Since(startTime).Milliseconds()

	// Extract token usage from response
	if response != nil && response.ResponseMeta != nil && response.ResponseMeta.Usage != nil {
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
