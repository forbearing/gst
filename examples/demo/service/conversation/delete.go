package conversation

import (
	"demo/model"

	"github.com/forbearing/gst/service"
	"github.com/forbearing/gst/types"
)

type Deleter struct {
	service.Base[*model.Conversation, *model.Conversation, *model.Conversation]
}

func (c *Deleter) Delete(ctx *types.ServiceContext, req *model.Conversation) (rsp *model.Conversation, err error) {
	log := c.WithServiceContext(ctx, ctx.GetPhase())
	log.Info("conversation delete")
	return rsp, nil
}

func (c *Deleter) DeleteBefore(ctx *types.ServiceContext, conversation *model.Conversation) error {
	log := c.WithServiceContext(ctx, ctx.GetPhase())
	log.Info("conversation delete before")
	return nil
}

func (c *Deleter) DeleteAfter(ctx *types.ServiceContext, conversation *model.Conversation) error {
	log := c.WithServiceContext(ctx, ctx.GetPhase())
	log.Info("conversation delete after")
	return nil
}
