package modeliam

import (
	"github.com/forbearing/gst/dsl"
	"github.com/forbearing/gst/model"
	"github.com/forbearing/gst/service"
)

type ChangePassword struct {
	model.Empty
}

type ChangePasswordReq struct {
	OldPassword string `json:"old_password" validate:"required"`
	NewPassword string `json:"new_password" validate:"required,min=6"`
}

type ChangePasswordRsp struct {
	Msg string `json:"msg"`
}

func (ChangePassword) Design() {
	dsl.Create(func() {
		dsl.Enabled(true)
		dsl.Service(true)
		dsl.Public(false)
		dsl.Payload[*ChangePasswordReq]()
		dsl.Result[*ChangePasswordRsp]()
	})
}

type ChangePasswordService struct {
	service.Base[*ChangePassword, *ChangePasswordReq, *ChangePasswordRsp]
}
