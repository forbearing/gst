package iam

import (
	modeliam "github.com/forbearing/gst/internal/model/iam"
	serviceiam "github.com/forbearing/gst/internal/service/iam"
	"github.com/forbearing/gst/types"
)

var _ types.Module[*EmailPasswordResetConfirm, *EmailPasswordResetConfirmReq, *EmailPasswordResetConfirmRsp] = (*EmailPasswordResetConfirmModule)(nil)

type (
	EmailPasswordResetConfirm        = modeliam.EmailPasswordResetConfirm
	EmailPasswordResetConfirmReq     = modeliam.EmailPasswordResetConfirmReq
	EmailPasswordResetConfirmRsp     = modeliam.EmailPasswordResetConfirmRsp
	EmailPasswordResetConfirmService = serviceiam.EmailPasswordResetConfirmService
	EmailPasswordResetConfirmModule  struct{}
)

func (*EmailPasswordResetConfirmModule) Service() types.Service[*EmailPasswordResetConfirm, *EmailPasswordResetConfirmReq, *EmailPasswordResetConfirmRsp] {
	return &EmailPasswordResetConfirmService{}
}

func (*EmailPasswordResetConfirmModule) Route() string { return "/iam/email-password-reset-confirm" }
func (*EmailPasswordResetConfirmModule) Pub() bool     { return true }
func (*EmailPasswordResetConfirmModule) Param() string { return "id" }
