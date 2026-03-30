package iam

import (
	modeliam "github.com/forbearing/gst/internal/model/iam"
	serviceiam "github.com/forbearing/gst/internal/service/iam"
	"github.com/forbearing/gst/types"
)

var _ types.Module[*EmailVerificationConfirm, *EmailVerificationConfirmReq, *EmailVerificationConfirmRsp] = (*EmailVerificationConfirmModule)(nil)

type (
	EmailVerificationConfirm        = modeliam.EmailVerificationConfirm
	EmailVerificationConfirmReq     = modeliam.EmailVerificationConfirmReq
	EmailVerificationConfirmRsp     = modeliam.EmailVerificationConfirmRsp
	EmailVerificationConfirmService = serviceiam.EmailVerificationConfirmService
	EmailVerificationConfirmModule  struct{}
)

func (*EmailVerificationConfirmModule) Service() types.Service[*EmailVerificationConfirm, *EmailVerificationConfirmReq, *EmailVerificationConfirmRsp] {
	return &EmailVerificationConfirmService{}
}

func (*EmailVerificationConfirmModule) Route() string { return "/iam/email-verification-confirm" }
func (*EmailVerificationConfirmModule) Pub() bool     { return true }
func (*EmailVerificationConfirmModule) Param() string { return "id" }
