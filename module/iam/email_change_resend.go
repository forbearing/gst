package iam

import (
	modeliamemail "github.com/forbearing/gst/internal/model/iam/email"
	serviceiamemail "github.com/forbearing/gst/internal/service/iam/email"
	"github.com/forbearing/gst/types"
)

var _ types.Module[*EmailChangeResend, *EmailChangeResendReq, *EmailChangeResendRsp] = (*EmailChangeResendModule)(nil)

type (
	EmailChangeResend        = modeliamemail.ChangeResend
	EmailChangeResendReq     = modeliamemail.ChangeResendReq
	EmailChangeResendRsp     = modeliamemail.ChangeResendRsp
	EmailChangeResendService = serviceiamemail.ChangeResendService
	EmailChangeResendModule  struct{}
)

func (*EmailChangeResendModule) Service() types.Service[*EmailChangeResend, *EmailChangeResendReq, *EmailChangeResendRsp] {
	return &EmailChangeResendService{}
}

func (*EmailChangeResendModule) Route() string { return "/iam/email/change-resend" }
func (*EmailChangeResendModule) Pub() bool     { return false }
func (*EmailChangeResendModule) Param() string { return "id" }
