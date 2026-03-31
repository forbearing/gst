package iam

import (
	modeliam "github.com/forbearing/gst/internal/model/iam"
	serviceiam "github.com/forbearing/gst/internal/service/iam"
	"github.com/forbearing/gst/types"
)

var _ types.Module[*EmailChangeCancel, *EmailChangeCancelReq, *EmailChangeCancelRsp] = (*EmailChangeCancelModule)(nil)

type (
	EmailChangeCancel        = modeliam.EmailChangeCancel
	EmailChangeCancelReq     = modeliam.EmailChangeCancelReq
	EmailChangeCancelRsp     = modeliam.EmailChangeCancelRsp
	EmailChangeCancelService = serviceiam.EmailChangeCancelService
	EmailChangeCancelModule  struct{}
)

func (*EmailChangeCancelModule) Service() types.Service[*EmailChangeCancel, *EmailChangeCancelReq, *EmailChangeCancelRsp] {
	return &EmailChangeCancelService{}
}

func (*EmailChangeCancelModule) Route() string { return "/iam/email-change-cancel" }
func (*EmailChangeCancelModule) Pub() bool     { return true }
func (*EmailChangeCancelModule) Param() string { return "id" }
