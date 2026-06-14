package model

import (
	"github.com/forbearing/gst/dsl"
	"github.com/forbearing/gst/model"
)

type Ping struct {
	model.Empty
}
type PingRsp struct {
	Msg string
}

func (Ping) Design() {
	dsl.List(func() {
		dsl.Public(true)
		dsl.Service(true)
		dsl.Result[*PingRsp]()
	})
}
