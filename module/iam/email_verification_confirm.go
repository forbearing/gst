package iam

import (
	modeliamemail "github.com/forbearing/gst/internal/model/iam/email"
	serviceiamemail "github.com/forbearing/gst/internal/service/iam/email"
	"github.com/forbearing/gst/types"
)

var _ types.Module[*EmailVerificationConfirm, *EmailVerificationConfirmReq, *EmailVerificationConfirmRsp] = (*EmailVerificationConfirmModule)(nil)

type (
	EmailVerificationConfirm        = modeliamemail.VerificationConfirm
	EmailVerificationConfirmReq     = modeliamemail.VerificationConfirmReq
	EmailVerificationConfirmRsp     = modeliamemail.VerificationConfirmRsp
	EmailVerificationConfirmService = serviceiamemail.VerificationConfirmService
	EmailVerificationConfirmModule  struct{}
)

func (*EmailVerificationConfirmModule) Service() types.Service[*EmailVerificationConfirm, *EmailVerificationConfirmReq, *EmailVerificationConfirmRsp] {
	return &EmailVerificationConfirmService{}
}

func (*EmailVerificationConfirmModule) Route() string { return "/iam/email/verification-confirm" }
func (*EmailVerificationConfirmModule) Pub() bool     { return true }
func (*EmailVerificationConfirmModule) Param() string { return "id" }
