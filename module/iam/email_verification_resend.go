package iam

import (
	modeliam "github.com/forbearing/gst/internal/model/iam"
	serviceiam "github.com/forbearing/gst/internal/service/iam"
	"github.com/forbearing/gst/types"
)

var _ types.Module[*EmailVerificationResend, *EmailVerificationResendReq, *EmailVerificationResendRsp] = (*EmailVerificationResendModule)(nil)

type (
	EmailVerificationResend        = modeliam.EmailVerificationResend
	EmailVerificationResendReq     = modeliam.EmailVerificationResendReq
	EmailVerificationResendRsp     = modeliam.EmailVerificationResendRsp
	EmailVerificationResendService = serviceiam.EmailVerificationResendService
	EmailVerificationResendModule  struct{}
)

func (*EmailVerificationResendModule) Service() types.Service[*EmailVerificationResend, *EmailVerificationResendReq, *EmailVerificationResendRsp] {
	return &EmailVerificationResendService{}
}

func (*EmailVerificationResendModule) Route() string { return "/iam/email-verification-resend" }
func (*EmailVerificationResendModule) Pub() bool     { return true }
func (*EmailVerificationResendModule) Param() string { return "id" }
