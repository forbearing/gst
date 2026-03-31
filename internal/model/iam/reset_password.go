package modeliam

import (
	"github.com/forbearing/gst/dsl"
	"github.com/forbearing/gst/model"
)

type ResetPassword struct {
	model.Empty
}

type ResetPasswordReq struct {
	UserID      string `json:"user_id" validate:"required"`
	NewPassword string `json:"new_password" validate:"required,min=6"`
}

type ResetPasswordRsp struct {
	Msg string `json:"msg,omitempty"`
}

func (ResetPassword) Design() {
	dsl.Create(func() {
		dsl.Enabled(true)
		dsl.Service(true)
		dsl.Public(false)
		dsl.Payload[*ResetPasswordReq]()
		dsl.Result[*ResetPasswordRsp]()
	})
}
