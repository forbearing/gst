package iam

import (
	modeliam "github.com/forbearing/gst/internal/model/iam"
	serviceiam "github.com/forbearing/gst/internal/service/iam"
	"github.com/forbearing/gst/types"
)

var _ types.Module[*EmailPasswordResetRequest, *EmailPasswordResetRequestReq, *EmailPasswordResetRequestRsp] = (*EmailPasswordResetRequestModule)(nil)

type (
	EmailPasswordResetRequest        = modeliam.EmailPasswordResetRequest
	EmailPasswordResetRequestReq     = modeliam.EmailPasswordResetRequestReq
	EmailPasswordResetRequestRsp     = modeliam.EmailPasswordResetRequestRsp
	EmailPasswordResetRequestService = serviceiam.EmailPasswordResetRequestService
	EmailPasswordResetRequestModule  struct{}
)

func (*EmailPasswordResetRequestModule) Service() types.Service[*EmailPasswordResetRequest, *EmailPasswordResetRequestReq, *EmailPasswordResetRequestRsp] {
	return &EmailPasswordResetRequestService{}
}

func (*EmailPasswordResetRequestModule) Route() string { return "/iam/email-password-reset-request" }
func (*EmailPasswordResetRequestModule) Pub() bool     { return true }
func (*EmailPasswordResetRequestModule) Param() string { return "id" }
