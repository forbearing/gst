package iam

import (
	modeliamemail "github.com/forbearing/gst/internal/model/iam/email"
	serviceiamemail "github.com/forbearing/gst/internal/service/iam/email"
	"github.com/forbearing/gst/types"
)

var _ types.Module[*EmailChangeCancel, *EmailChangeCancelReq, *EmailChangeCancelRsp] = (*EmailChangeCancelModule)(nil)

type (
	EmailChangeCancel       = modeliamemail.ChangeCancel
	EmailChangeCancelReq    = modeliamemail.ChangeCancelReq
	EmailChangeCancelRsp    = modeliamemail.ChangeCancelRsp
	EmailChangeCancelModule struct{}
)

func (*EmailChangeCancelModule) Service() types.Service[*EmailChangeCancel, *EmailChangeCancelReq, *EmailChangeCancelRsp] {
	return &serviceiamemail.ChangeCancelService{}
}

func (*EmailChangeCancelModule) Route() string { return "/iam/email/change-cancel" }
func (*EmailChangeCancelModule) Pub() bool     { return true }
func (*EmailChangeCancelModule) Param() string { return "id" }
