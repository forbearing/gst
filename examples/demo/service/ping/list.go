package ping

import (
	"demo/model"

	"github.com/forbearing/gst/service"
	"github.com/forbearing/gst/types"
)

type Lister struct {
	service.Base[*model.Ping, *model.Ping, *model.PingRsp]
}

func (p *Lister) List(ctx *types.ServiceContext, req *model.Ping) (rsp *model.PingRsp, err error) {
	return &model.PingRsp{
		Msg: "pong",
	}, nil
}
