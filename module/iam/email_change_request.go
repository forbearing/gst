package iam

import (
	modeliam "github.com/forbearing/gst/internal/model/iam"
	serviceiam "github.com/forbearing/gst/internal/service/iam"
	"github.com/forbearing/gst/types"
)

var _ types.Module[*EmailChangeRequest, *EmailChangeRequestReq, *EmailChangeRequestRsp] = (*EmailChangeRequestModule)(nil)

type (
	EmailChangeRequest        = modeliam.EmailChangeRequest
	EmailChangeRequestReq     = modeliam.EmailChangeRequestReq
	EmailChangeRequestRsp     = modeliam.EmailChangeRequestRsp
	EmailChangeRequestService = serviceiam.EmailChangeRequestService
	EmailChangeRequestModule  struct{}
)

func (*EmailChangeRequestModule) Service() types.Service[*EmailChangeRequest, *EmailChangeRequestReq, *EmailChangeRequestRsp] {
	return &EmailChangeRequestService{}
}

func (*EmailChangeRequestModule) Route() string { return "/iam/email-change-request" }
func (*EmailChangeRequestModule) Pub() bool     { return false }
func (*EmailChangeRequestModule) Param() string { return "id" }
