package iam

import (
	modeliam "github.com/forbearing/gst/internal/model/iam"
	serviceiam "github.com/forbearing/gst/internal/service/iam"
	"github.com/forbearing/gst/types"
)

var _ types.Module[*AccountStatus, *AccountStatusReq, *AccountStatusRsp] = (*AccountStatusModule)(nil)

type (
	AccountStatus       = modeliam.AccountStatus
	AccountStatusReq    = modeliam.AccountStatusReq
	AccountStatusRsp    = modeliam.AccountStatusRsp
	AccountStatusModule struct{}
)

func (*AccountStatusModule) Service() types.Service[*AccountStatus, *AccountStatusReq, *AccountStatusRsp] {
	return &serviceiam.AccountStatusService{}
}

func (*AccountStatusModule) Route() string { return "/iam/account-status" }
func (*AccountStatusModule) Pub() bool     { return false }
func (*AccountStatusModule) Param() string { return "id" }
