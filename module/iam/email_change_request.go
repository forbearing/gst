package iam

import (
	modeliamemail "github.com/forbearing/gst/internal/model/iam/email"
	serviceiamemail "github.com/forbearing/gst/internal/service/iam/email"
	"github.com/forbearing/gst/types"
)

var _ types.Module[*EmailChangeRequest, *EmailChangeRequestReq, *EmailChangeRequestRsp] = (*EmailChangeRequestModule)(nil)

type (
	EmailChangeRequest       = modeliamemail.ChangeRequest
	EmailChangeRequestReq    = modeliamemail.ChangeRequestReq
	EmailChangeRequestRsp    = modeliamemail.ChangeRequestRsp
	EmailChangeRequestModule struct{}
)

func (*EmailChangeRequestModule) Service() types.Service[*EmailChangeRequest, *EmailChangeRequestReq, *EmailChangeRequestRsp] {
	return &serviceiamemail.ChangeRequestService{}
}

func (*EmailChangeRequestModule) Route() string { return "/iam/email/change-request" }
func (*EmailChangeRequestModule) Pub() bool     { return false }
func (*EmailChangeRequestModule) Param() string { return "id" }
