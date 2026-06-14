package ping

import (
	"demo/model"

	"github.com/forbearing/gst/database"
	"github.com/forbearing/gst/module/iam"
	"github.com/forbearing/gst/service"
	"github.com/forbearing/gst/types"
)

type Lister struct {
	service.Base[*model.Ping, *model.Ping, *model.PingRsp]
}

func (p *Lister) List(ctx *types.ServiceContext, req *model.Ping) (rsp *model.PingRsp, err error) {
	users := make([]*iam.User, 0)
	n := new(int64)
	_ = database.Database[*iam.User](ctx.DatabaseContext()).WithDryRun().List(&users)
	_ = database.Database[*iam.User](ctx.DatabaseContext()).WithDryRun().Count(n)

	return &model.PingRsp{
		Msg: "pong",
	}, nil
}
