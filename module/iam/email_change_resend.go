package iam

import (
	modeliam "github.com/forbearing/gst/internal/model/iam"
	serviceiam "github.com/forbearing/gst/internal/service/iam"
	"github.com/forbearing/gst/types"
)

var _ types.Module[*EmailChangeResend, *EmailChangeResendReq, *EmailChangeResendRsp] = (*EmailChangeResendModule)(nil)

type (
	EmailChangeResend        = modeliam.EmailChangeResend
	EmailChangeResendReq     = modeliam.EmailChangeResendReq
	EmailChangeResendRsp     = modeliam.EmailChangeResendRsp
	EmailChangeResendService = serviceiam.EmailChangeResendService
	EmailChangeResendModule  struct{}
)

func (*EmailChangeResendModule) Service() types.Service[*EmailChangeResend, *EmailChangeResendReq, *EmailChangeResendRsp] {
	return &EmailChangeResendService{}
}

func (*EmailChangeResendModule) Route() string { return "/iam/email-change-resend" }
func (*EmailChangeResendModule) Pub() bool     { return false }
func (*EmailChangeResendModule) Param() string { return "id" }
