package iam

import (
	modeliamemail "github.com/forbearing/gst/internal/model/iam/email"
	serviceiamemail "github.com/forbearing/gst/internal/service/iam/email"
	"github.com/forbearing/gst/types"
)

var _ types.Module[*EmailPasswordResetConfirm, *EmailPasswordResetConfirmReq, *EmailPasswordResetConfirmRsp] = (*EmailPasswordResetConfirmModule)(nil)

type (
	EmailPasswordResetConfirm       = modeliamemail.PasswordResetConfirm
	EmailPasswordResetConfirmReq    = modeliamemail.PasswordResetConfirmReq
	EmailPasswordResetConfirmRsp    = modeliamemail.PasswordResetConfirmRsp
	EmailPasswordResetConfirmModule struct{}
)

func (*EmailPasswordResetConfirmModule) Service() types.Service[*EmailPasswordResetConfirm, *EmailPasswordResetConfirmReq, *EmailPasswordResetConfirmRsp] {
	return &serviceiamemail.PasswordResetConfirmService{}
}

func (*EmailPasswordResetConfirmModule) Route() string { return "/iam/email/password-reset-confirm" }
func (*EmailPasswordResetConfirmModule) Pub() bool     { return true }
func (*EmailPasswordResetConfirmModule) Param() string { return "id" }
