package iam

import (
	modeliam "github.com/forbearing/gst/internal/model/iam"
	serviceiam "github.com/forbearing/gst/internal/service/iam"
	"github.com/forbearing/gst/types"
)

var _ types.Module[*ChangePassword, *ChangePasswordReq, *ChangePasswordRsp] = (*ChangePasswordModule)(nil)

type (
	ChangePassword        = modeliam.ChangePassword
	ChangePasswordReq     = modeliam.ChangePasswordReq
	ChangePasswordRsp     = modeliam.ChangePasswordRsp
	ChangePasswordService = serviceiam.ChangePasswordService
	ChangePasswordModule  struct{}
)

func (*ChangePasswordModule) Service() types.Service[*ChangePassword, *ChangePasswordReq, *ChangePasswordRsp] {
	return &ChangePasswordService{}
}
func (*ChangePasswordModule) Route() string { return "/iam/change-password" }
func (*ChangePasswordModule) Pub() bool     { return false }
func (*ChangePasswordModule) Param() string { return "id" }
