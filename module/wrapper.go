package module

import (
	"github.com/forbearing/gst/model"
	"github.com/forbearing/gst/service"
	"github.com/forbearing/gst/types"
)

var _ types.Module[*model.Empty, *model.Empty, *model.Empty] = &Wrapper[*model.Empty, *model.Empty, *model.Empty]{}

type Wrapper[M types.Model, REQ types.Request, RSP types.Response] struct {
	route string
	param string
	pub   bool
}

func (w *Wrapper[M, REQ, RSP]) Service() types.Service[M, REQ, RSP] {
	return &service.Base[M, REQ, RSP]{}
}

func (w *Wrapper[M, REQ, RSP]) Route() string {
	return w.route
}

func (w *Wrapper[M, REQ, RSP]) Pub() bool {
	return w.pub
}

func (w *Wrapper[M, REQ, RSP]) Param() string {
	return w.param
}

func NewWrapper[M types.Model, REQ types.Request, RSP types.Response](route string, param string, pub bool) types.Module[M, REQ, RSP] {
	if len(param) == 0 {
		param = "id"
	}
	return &Wrapper[M, REQ, RSP]{
		route: route,
		param: param,
		pub:   pub,
	}
}
