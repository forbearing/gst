package iam

import (
	modeliamemail "github.com/forbearing/gst/internal/model/iam/email"
	serviceiamemail "github.com/forbearing/gst/internal/service/iam/email"
	"github.com/forbearing/gst/types"
)

var _ types.Module[*EmailChangeConfirm, *EmailChangeConfirmReq, *EmailChangeConfirmRsp] = (*EmailChangeConfirmModule)(nil)

type (
	EmailChangeConfirm       = modeliamemail.ChangeConfirm
	EmailChangeConfirmReq    = modeliamemail.ChangeConfirmReq
	EmailChangeConfirmRsp    = modeliamemail.ChangeConfirmRsp
	EmailChangeConfirmModule struct{}
)

func (*EmailChangeConfirmModule) Service() types.Service[*EmailChangeConfirm, *EmailChangeConfirmReq, *EmailChangeConfirmRsp] {
	return &serviceiamemail.ChangeConfirmService{}
}

func (*EmailChangeConfirmModule) Route() string { return "/iam/email/change-confirm" }
func (*EmailChangeConfirmModule) Pub() bool     { return false }
func (*EmailChangeConfirmModule) Param() string { return "id" }
