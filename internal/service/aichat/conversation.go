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

type Conversation struct {
	service.Base[*modelaichat.Conversation, *modelaichat.Conversation, *modelaichat.Conversation]
}

// deleteConversationMessages deletes all messages and related feedbacks for a conversation within a transaction
func deleteConversationMessages(tx any, conversationID string, log types.Logger) error {
	// Get all messages in this conversation within the transaction
	messages := make([]*modelaichat.Message, 0)
	if err := database.Database[*modelaichat.Message](nil).
		WithTx(tx).
		WithQuery(&modelaichat.Message{ConversationID: conversationID}).
		List(&messages); err != nil {
		return errors.Wrapf(err, "failed to get messages for conversation: %s", conversationID)
	}

	if len(messages) == 0 {
		return nil
	}

	// Collect all message IDs for deleting related feedbacks
	messageIDs := make([]string, 0, len(messages))
	for _, msg := range messages {
		messageIDs = append(messageIDs, msg.ID)
	}

	// Delete all feedbacks related to these messages
	if len(messageIDs) > 0 {
		feedbacks := make([]*modelaichat.MessageFeedback, 0)
		if err := database.Database[*modelaichat.MessageFeedback](nil).
			WithTx(tx).
			WithQuery(&modelaichat.MessageFeedback{MessageID: strings.Join(messageIDs, ",")}).
			List(&feedbacks); err != nil {
			return errors.Wrapf(err, "failed to get message feedbacks for conversation: %s", conversationID)
		}
		if err := database.Database[*modelaichat.MessageFeedback](nil).WithTx(tx).Delete(feedbacks...); err != nil {
			return errors.Wrapf(err, "failed to delete message feedbacks for conversation: %s", conversationID)
		}
		if log != nil {
			log.Infow("deleted message feedbacks", "conversation_id", conversationID, "count", len(feedbacks))
		}
	}

	// Delete all messages related to this conversation
	if err := database.Database[*modelaichat.Message](nil).WithTx(tx).Delete(messages...); err != nil {
		return errors.Wrapf(err, "failed to delete messages for conversation: %s", conversationID)
	}
	if log != nil {
		log.Infow("deleted messages", "conversation_id", conversationID, "count", len(messages))
	}

	return nil
}

// DeleteBefore deletes all messages and related feedbacks for a conversation before the conversation is deleted
func (s *Conversation) DeleteBefore(ctx *types.ServiceContext, conv *modelaichat.Conversation) error {
	log := s.WithServiceContext(ctx, ctx.GetPhase())
	log.Infow("delete conversation before", "conversation_id", conv.ID)

	// Delete all messages and feedbacks within a transaction
	err := database.Database[*model.Any](ctx.DatabaseContext()).TransactionFunc(func(tx any) error {
		return deleteConversationMessages(tx, conv.ID, log)
	})
	if err != nil {
		return errors.Wrap(err, "failed to delete conversation messages and feedbacks")
	}

	return nil
}

// DeleteManyBefore deletes all messages and related feedbacks for multiple conversations before they are deleted
func (s *Conversation) DeleteManyBefore(ctx *types.ServiceContext, convs ...*modelaichat.Conversation) error {
	log := s.WithServiceContext(ctx, ctx.GetPhase())
	log.Infow("delete many conversations before", "count", len(convs))

	// Delete all messages and feedbacks for each conversation within a transaction
	err := database.Database[*model.Any](ctx.DatabaseContext()).TransactionFunc(func(tx any) error {
		for _, conv := range convs {
			if err := deleteConversationMessages(tx, conv.ID, log); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return errors.Wrap(err, "failed to delete conversations messages and feedbacks")
	}

	return nil
}
