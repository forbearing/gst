package serviceaichat

import (
	"github.com/cockroachdb/errors"
	"github.com/forbearing/gst/database"
	modelaichat "github.com/forbearing/gst/internal/model/aichat"
	"github.com/forbearing/gst/model"
	"github.com/forbearing/gst/service"
	"github.com/forbearing/gst/types"
	"github.com/forbearing/gst/util"
)

// StopMessage handles stopping a streaming message
type StopMessage struct {
	service.Base[*model.Empty, *modelaichat.StopMessageReq, *modelaichat.StopMessageRsp]
}

func (s *StopMessage) Create(ctx *types.ServiceContext, req *modelaichat.StopMessageReq) (*modelaichat.StopMessageRsp, error) {
	log := s.WithServiceContext(ctx, ctx.GetPhase())

	// Validate request
	if len(req.MessageID) == 0 {
		return nil, errors.New("message_id is required")
	}

	// Get message from database
	msg := new(modelaichat.Message)
	if err := database.Database[*modelaichat.Message](ctx.DatabaseContext()).
		WithQuery(&modelaichat.Message{Base: model.Base{ID: req.MessageID}}).
		First(msg); err != nil {
		return nil, errors.Wrapf(err, "failed to get message: %s", req.MessageID)
	}

	// Verify message belongs to current user
	if msg.ConversationID != "" {
		conversation := new(modelaichat.Conversation)
		if err := database.Database[*modelaichat.Conversation](ctx.DatabaseContext()).
			WithQuery(&modelaichat.Conversation{Base: model.Base{ID: msg.ConversationID}}).
			First(conversation); err == nil {
			if conversation.UserID != ctx.UserID {
				return nil, errors.New("message does not belong to current user")
			}
		}
	}

	// Check if message is in streaming status
	if msg.Status != modelaichat.MessageStatusStreaming {
		return nil, errors.Newf("message is not in streaming status, current status: %s", msg.Status)
	}

	// Cancel the stream
	streamManager := GetStreamManager()
	if err := streamManager.CancelStream(req.MessageID); err != nil {
		log.Warnw("failed to cancel stream", "error", err, "message_id", req.MessageID)
		// Even if cancel fails, try to update message status
	}

	// Update message status regardless of cancel result
	msg.Status = modelaichat.MessageStatusStopped
	msg.StopReason = util.ValueOf(modelaichat.StopReasonUser)
	if err := database.Database[*modelaichat.Message](ctx.DatabaseContext()).Update(msg); err != nil {
		return nil, errors.Wrap(err, "failed to update message status")
	}

	log.Infow("message stopped", "message_id", req.MessageID)
	return &modelaichat.StopMessageRsp{MessageID: req.MessageID, Content: msg.Content}, nil
}
