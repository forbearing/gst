package iam

import (
	modeliam "github.com/forbearing/gst/internal/model/iam"
	serviceiam "github.com/forbearing/gst/internal/service/iam"
	"github.com/forbearing/gst/types"
)

var _ types.Module[*ResetPassword, *ResetPasswordReq, *ResetPasswordRsp] = (*ResetPasswordModule)(nil)

type (
	ResetPassword       = modeliam.ResetPassword
	ResetPasswordReq    = modeliam.ResetPasswordReq
	ResetPasswordRsp    = modeliam.ResetPasswordRsp
	ResetPasswordModule struct{}
)

func (*ResetPasswordModule) Service() types.Service[*ResetPassword, *ResetPasswordReq, *ResetPasswordRsp] {
	return &serviceiam.ResetPasswordService{}
}
func (*ResetPasswordModule) Route() string { return "/iam/reset-password" }
func (*ResetPasswordModule) Pub() bool     { return false }
func (*ResetPasswordModule) Param() string { return "id" }
