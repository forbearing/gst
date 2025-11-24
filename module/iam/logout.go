package iam

import (
	modeliam "github.com/forbearing/gst/internal/model/iam"
	serviceiam "github.com/forbearing/gst/internal/service/iam"
	"github.com/forbearing/gst/types"
)

var _ types.Module[*Logout, *Logout, *LogoutRsp] = (*LogoutModule)(nil)

type (
	Logout        = modeliam.Logout
	LogoutRsp     = modeliam.LogoutRsp
	LogoutService = serviceiam.LogoutService
	LogoutModule  struct{}
)

func (*LogoutModule) Service() types.Service[*Logout, *Logout, *LogoutRsp] {
	return &LogoutService{}
}
func (*LogoutModule) Route() string { return "/iam/logout" }
func (*LogoutModule) Pub() bool     { return false }
func (*LogoutModule) Param() string { return "id" }
