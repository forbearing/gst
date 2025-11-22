package modeliam

import (
	"github.com/forbearing/gst/dsl"
	"github.com/forbearing/gst/model"
)

type Logout struct {
	model.Empty
}

type LogoutRsp struct {
	Msg string `json:"msg"`
}

func (Logout) Design() {
	dsl.Create(func() {
		dsl.Enabled(true)
		dsl.Service(true)
		dsl.Public(true)
		dsl.Result[*LogoutRsp]()
	})
}
