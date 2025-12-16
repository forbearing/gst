package serviceaichat

import (
	"strings"

	"github.com/cockroachdb/errors"
	"github.com/forbearing/gst/database"
	modelaichat "github.com/forbearing/gst/internal/model/aichat"
	"github.com/forbearing/gst/model"
	"github.com/forbearing/gst/service"
	"github.com/forbearing/gst/types"
)

// ClearConversationMessages handles clearing all messages in a conversation while keeping the conversation itself
type ClearConversationMessages struct {
	service.Base[*model.Empty, *modelaichat.ClearConversationMessagesReq, *modelaichat.ClearConversationMessagesRsp]
}

// Create clears all messages in the specified conversation
func (s *ClearConversationMessages) Create(ctx *types.ServiceContext, req *modelaichat.ClearConversationMessagesReq) (*modelaichat.ClearConversationMessagesRsp, error) {
	log := s.WithServiceContext(ctx, ctx.GetPhase())
	log.Infow("clear conversation messages", "conversation_id", req.ConversationID)

	// Validate request
	if len(req.ConversationID) == 0 {
		return nil, errors.New("conversation_id is required")
	}

	// Get conversation to verify it exists and belongs to current user
	conversation := new(modelaichat.Conversation)
	if err := database.Database[*modelaichat.Conversation](ctx.DatabaseContext()).Get(conversation, req.ConversationID); err != nil {
		return nil, errors.Wrapf(err, "failed to get conversation: %s", req.ConversationID)
	}

	// Verify conversation belongs to current user
	if conversation.UserID != ctx.UserID {
		return nil, errors.New("conversation does not belong to current user")
	}

	// Delete all messages within a transaction to avoid race conditions
	var deletedCount int
	err := database.Database[*model.Any](ctx.DatabaseContext()).TransactionFunc(func(tx any) error {
		// Get all messages in this conversation within the transaction
		messages := make([]*modelaichat.Message, 0)
		if err := database.Database[*modelaichat.Message](ctx.DatabaseContext()).
			WithTx(tx).
			WithQuery(&modelaichat.Message{ConversationID: req.ConversationID}).
			List(&messages); err != nil {
			return errors.Wrap(err, "failed to get messages")
		}

		deletedCount = len(messages)

		if deletedCount > 0 {
			// Collect all message IDs for deleting related feedbacks
			messageIDs := make([]string, 0, len(messages))
			for _, msg := range messages {
				messageIDs = append(messageIDs, msg.ID)
			}

			// Delete all feedbacks related to these messages
			if len(messageIDs) > 0 {
				feedbacks := make([]*modelaichat.MessageFeedback, 0)
				if err := database.Database[*modelaichat.MessageFeedback](ctx.DatabaseContext()).
					WithTx(tx).
					WithQuery(&modelaichat.MessageFeedback{MessageID: strings.Join(messageIDs, ",")}).
					List(&feedbacks); err != nil {
					return errors.Wrap(err, "failed to get message feedbacks")
				}
				if err := database.Database[*modelaichat.MessageFeedback](ctx.DatabaseContext()).WithTx(tx).Delete(feedbacks...); err != nil {
					return errors.Wrap(err, "failed to delete message feedbacks")
				}
				log.Infow("deleted message feedbacks", "count", len(feedbacks))
			}

			// Delete all messages related to this conversation.
			if err := database.Database[*modelaichat.Message](ctx.DatabaseContext()).WithTx(tx).Delete(messages...); err != nil {
				return errors.Wrap(err, "failed to delete messages")
			}
		}

		// Update conversation statistics
		conversation.MessageCount = 0
		conversation.TokensUsed = 0
		if err := database.Database[*modelaichat.Conversation](ctx.DatabaseContext()).WithTx(tx).Update(conversation); err != nil {
			return errors.Wrap(err, "failed to update conversation statistics")
		}
		return nil
	})
	if err != nil {
		return nil, errors.Wrap(err, "failed to clear conversation messages")
	}

	log.Infow("conversation messages cleared", "conversation_id", req.ConversationID, "deleted_count", deletedCount)
	return &modelaichat.ClearConversationMessagesRsp{
		ConversationID: req.ConversationID,
		DeletedCount:   deletedCount,
	}, nil
}
