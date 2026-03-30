package iam

import (
	modeliam "github.com/forbearing/gst/internal/model/iam"
	serviceiam "github.com/forbearing/gst/internal/service/iam"
	"github.com/forbearing/gst/types"
)

var _ types.Module[*EmailChangeConfirm, *EmailChangeConfirmReq, *EmailChangeConfirmRsp] = (*EmailChangeConfirmModule)(nil)

type (
	EmailChangeConfirm        = modeliam.EmailChangeConfirm
	EmailChangeConfirmReq     = modeliam.EmailChangeConfirmReq
	EmailChangeConfirmRsp     = modeliam.EmailChangeConfirmRsp
	EmailChangeConfirmService = serviceiam.EmailChangeConfirmService
	EmailChangeConfirmModule  struct{}
)

func (*EmailChangeConfirmModule) Service() types.Service[*EmailChangeConfirm, *EmailChangeConfirmReq, *EmailChangeConfirmRsp] {
	return &EmailChangeConfirmService{}
}

func (*EmailChangeConfirmModule) Route() string { return "/iam/email-change-confirm" }
func (*EmailChangeConfirmModule) Pub() bool     { return false }
func (*EmailChangeConfirmModule) Param() string { return "id" }
