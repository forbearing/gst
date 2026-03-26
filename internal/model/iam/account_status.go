package modeliam

import (
	"github.com/forbearing/gst/dsl"
	"github.com/forbearing/gst/model"
)

type AccountStatus struct {
	model.Empty
}

type AccountStatusReq struct {
	UserID string     `json:"user_id" validate:"required"`
	Status UserStatus `json:"status" validate:"required"`
}

type AccountStatusRsp struct {
	Msg string `json:"msg,omitempty"`
}

func (AccountStatus) Design() {
	dsl.Create(func() {
		dsl.Enabled(true)
		dsl.Service(true)
		dsl.Public(false)
		dsl.Payload[*AccountStatusReq]()
		dsl.Result[*AccountStatusRsp]()
	})
}
