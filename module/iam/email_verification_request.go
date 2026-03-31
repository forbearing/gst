package iam

import (
	modeliamemail "github.com/forbearing/gst/internal/model/iam/email"
	serviceiamemail "github.com/forbearing/gst/internal/service/iam/email"
	"github.com/forbearing/gst/types"
)

var _ types.Module[*EmailVerificationRequest, *EmailVerificationRequestReq, *EmailVerificationRequestRsp] = (*EmailVerificationRequestModule)(nil)

type (
	EmailVerificationRequest       = modeliamemail.VerificationRequest
	EmailVerificationRequestReq    = modeliamemail.VerificationRequestReq
	EmailVerificationRequestRsp    = modeliamemail.VerificationRequestRsp
	EmailVerificationRequestModule struct{}
)

func (*EmailVerificationRequestModule) Service() types.Service[*EmailVerificationRequest, *EmailVerificationRequestReq, *EmailVerificationRequestRsp] {
	return &serviceiamemail.VerificationRequestService{}
}

func (*EmailVerificationRequestModule) Route() string { return "/iam/email/verification-request" }
func (*EmailVerificationRequestModule) Pub() bool     { return true }
func (*EmailVerificationRequestModule) Param() string { return "id" }
