package iam

import (
	modeliam "github.com/forbearing/gst/internal/model/iam"
	serviceiam "github.com/forbearing/gst/internal/service/iam"
	"github.com/forbearing/gst/types"
)

var _ types.Module[*EmailVerificationRequest, *EmailVerificationRequestReq, *EmailVerificationRequestRsp] = (*EmailVerificationRequestModule)(nil)

type (
	EmailVerificationRequest        = modeliam.EmailVerificationRequest
	EmailVerificationRequestReq     = modeliam.EmailVerificationRequestReq
	EmailVerificationRequestRsp     = modeliam.EmailVerificationRequestRsp
	EmailVerificationRequestService = serviceiam.EmailVerificationRequestService
	EmailVerificationRequestModule  struct{}
)

func (*EmailVerificationRequestModule) Service() types.Service[*EmailVerificationRequest, *EmailVerificationRequestReq, *EmailVerificationRequestRsp] {
	return &EmailVerificationRequestService{}
}

func (*EmailVerificationRequestModule) Route() string { return "/iam/email-verification-request" }
func (*EmailVerificationRequestModule) Pub() bool     { return true }
func (*EmailVerificationRequestModule) Param() string { return "id" }
