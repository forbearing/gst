package iam

import (
	modeliam "github.com/forbearing/gst/internal/model/iam"
	serviceiam "github.com/forbearing/gst/internal/service/iam"
	"github.com/forbearing/gst/types"
)

var _ types.Module[*Login, *LoginReq, *LoginRsp] = (*LoginModule)(nil)

type (
	Login        = modeliam.Login
	LoginReq     = modeliam.LoginReq
	LoginRsp     = modeliam.LoginRsp
	LoginService = serviceiam.LoginService
	LoginModule  struct{}
)

func (*LoginModule) Service() types.Service[*Login, *LoginReq, *LoginRsp] {
	return &LoginService{}
}
func (*LoginModule) Route() string { return "/login" }
func (*LoginModule) Pub() bool     { return true }
func (*LoginModule) Param() string { return "id" }
