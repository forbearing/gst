package serviceaichat

import (
	"github.com/cockroachdb/errors"
	"github.com/forbearing/gst/database"
	modelaichat "github.com/forbearing/gst/internal/model/aichat"
	"github.com/forbearing/gst/model"
	"github.com/forbearing/gst/service"
	"github.com/forbearing/gst/types"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

// SubmitMessageFeedback handles submitting feedback for a message
type SubmitMessageFeedback struct {
	service.Base[*model.Empty, *modelaichat.SubmitMessageFeedbackReq, *modelaichat.SubmitMessageFeedbackRsp]
}

func (s *SubmitMessageFeedback) Create(ctx *types.ServiceContext, req *modelaichat.SubmitMessageFeedbackReq) (*modelaichat.SubmitMessageFeedbackRsp, error) {
	log := s.WithServiceContext(ctx, ctx.GetPhase())
	log.Infow("submit message feedback", "message_id", req.MessageID, "type", req.Type)

	// Validate request
	if len(req.MessageID) == 0 {
		return nil, errors.New("message_id is required")
	}
	if req.Type != modelaichat.FeedbackLike && req.Type != modelaichat.FeedbackDislike {
		return nil, errors.New("type must be 'like' or 'dislike'")
	}

	// Get message from database
	msg := new(modelaichat.Message)
	if err := database.Database[*modelaichat.Message](ctx.DatabaseContext()).Get(msg, req.MessageID); err != nil {
		return nil, errors.Wrapf(err, "failed to get message: %s", req.MessageID)
	}

	// Verify it's an assistant message
	if msg.Role != modelaichat.MessageRoleAssistant {
		return nil, errors.New("can only submit feedback for assistant messages")
	}

	// Verify message status allows feedback (only completed, stopped, or failed messages can receive feedback)
	if msg.Status != modelaichat.MessageStatusCompleted &&
		msg.Status != modelaichat.MessageStatusStopped &&
		msg.Status != modelaichat.MessageStatusFailed {
		return nil, errors.Newf("cannot submit feedback for message with status: %s", msg.Status)
	}

	// Verify message is active
	if msg.IsActive == nil || !*msg.IsActive {
		return nil, errors.New("can only submit feedback for active messages")
	}

	// Verify message belongs to current user
	if msg.ConversationID == "" {
		return nil, errors.New("message conversation_id is empty")
	}
	conversation := new(modelaichat.Conversation)
	if err := database.Database[*modelaichat.Conversation](ctx.DatabaseContext()).Get(conversation, msg.ConversationID); err != nil {
		return nil, errors.Wrapf(err, "failed to get conversation: %s", msg.ConversationID)
	}
	if conversation.UserID != ctx.UserID {
		return nil, errors.New("message does not belong to current user")
	}

	// Check if feedback already exists for this message and user
	existingFeedback := new(modelaichat.MessageFeedback)
	err := database.Database[*modelaichat.MessageFeedback](ctx.DatabaseContext()).
		WithQuery(&modelaichat.MessageFeedback{
			MessageID: req.MessageID,
			UserID:    ctx.UserID,
		}).
		First(existingFeedback)
	feedbackExists := err == nil
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, errors.Wrap(err, "failed to check existing feedback")
	}

	// Prepare feedback data
	var categories datatypes.JSONSlice[modelaichat.FeedbackCategory]
	if len(req.Categories) > 0 {
		categories = datatypes.JSONSlice[modelaichat.FeedbackCategory](req.Categories)
	}

	if feedbackExists {
		// Update existing feedback
		existingFeedback.Type = req.Type
		existingFeedback.Categories = categories
		existingFeedback.Comment = req.Comment
		existingFeedback.ExpectedAnswer = req.ExpectedAnswer

		if err := database.Database[*modelaichat.MessageFeedback](ctx.DatabaseContext()).Update(existingFeedback); err != nil {
			return nil, errors.Wrap(err, "failed to update feedback")
		}

		log.Infow("feedback updated", "feedback_id", existingFeedback.ID, "message_id", req.MessageID)
		return &modelaichat.SubmitMessageFeedbackRsp{
			FeedbackID: existingFeedback.ID,
			MessageID:  req.MessageID,
		}, nil
	}

	// Create new feedback
	newFeedback := &modelaichat.MessageFeedback{
		MessageID:      req.MessageID,
		UserID:         ctx.UserID,
		Type:           req.Type,
		Categories:     categories,
		Comment:        req.Comment,
		ExpectedAnswer: req.ExpectedAnswer,
	}

	if err := database.Database[*modelaichat.MessageFeedback](ctx.DatabaseContext()).Create(newFeedback); err != nil {
		return nil, errors.Wrap(err, "failed to create feedback")
	}

	log.Infow("feedback created", "feedback_id", newFeedback.ID, "message_id", req.MessageID)
	return &modelaichat.SubmitMessageFeedbackRsp{
		FeedbackID: newFeedback.ID,
		MessageID:  req.MessageID,
	}, nil
}
