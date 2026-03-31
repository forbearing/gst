package iam

import (
	modeliamemail "github.com/forbearing/gst/internal/model/iam/email"
	serviceiamemail "github.com/forbearing/gst/internal/service/iam/email"
	"github.com/forbearing/gst/types"
)

var _ types.Module[*EmailPasswordResetRequest, *EmailPasswordResetRequestReq, *EmailPasswordResetRequestRsp] = (*EmailPasswordResetRequestModule)(nil)

type (
	EmailPasswordResetRequest        = modeliamemail.PasswordResetRequest
	EmailPasswordResetRequestReq     = modeliamemail.PasswordResetRequestReq
	EmailPasswordResetRequestRsp     = modeliamemail.PasswordResetRequestRsp
	EmailPasswordResetRequestService = serviceiamemail.PasswordResetRequestService
	EmailPasswordResetRequestModule  struct{}
)

func (*EmailPasswordResetRequestModule) Service() types.Service[*EmailPasswordResetRequest, *EmailPasswordResetRequestReq, *EmailPasswordResetRequestRsp] {
	return &EmailPasswordResetRequestService{}
}

func (*EmailPasswordResetRequestModule) Route() string { return "/iam/email/password-reset-request" }
func (*EmailPasswordResetRequestModule) Pub() bool     { return true }
func (*EmailPasswordResetRequestModule) Param() string { return "id" }
