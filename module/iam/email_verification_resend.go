package iam

import (
	modeliamemail "github.com/forbearing/gst/internal/model/iam/email"
	serviceiamemail "github.com/forbearing/gst/internal/service/iam/email"
	"github.com/forbearing/gst/types"
)

var _ types.Module[*EmailVerificationResend, *EmailVerificationResendReq, *EmailVerificationResendRsp] = (*EmailVerificationResendModule)(nil)

type (
	EmailVerificationResend       = modeliamemail.VerificationResend
	EmailVerificationResendReq    = modeliamemail.VerificationResendReq
	EmailVerificationResendRsp    = modeliamemail.VerificationResendRsp
	EmailVerificationResendModule struct{}
)

func (*EmailVerificationResendModule) Service() types.Service[*EmailVerificationResend, *EmailVerificationResendReq, *EmailVerificationResendRsp] {
	return &serviceiamemail.VerificationResendService{}
}

func (*EmailVerificationResendModule) Route() string { return "/iam/email/verification-resend" }
func (*EmailVerificationResendModule) Pub() bool     { return true }
func (*EmailVerificationResendModule) Param() string { return "id" }
