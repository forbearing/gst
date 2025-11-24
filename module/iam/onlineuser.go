package iam

import (
	modeliam "github.com/forbearing/gst/internal/model/iam"
	serviceiam "github.com/forbearing/gst/internal/service/iam"
	"github.com/forbearing/gst/types"
)

var _ types.Module[*OnlineUser, *OnlineUser, *OnlineUser] = (*OnlineUserModule)(nil)

type (
	OnlineUser        = modeliam.OnlineUser
	OnlineUserService = serviceiam.OnlineUserService
	OnlineUserModule  struct{}
)

func (*OnlineUserModule) Service() types.Service[*OnlineUser, *OnlineUser, *OnlineUser] {
	return &OnlineUserService{}
}
func (*OnlineUserModule) Route() string { return "/online-users" }
func (*OnlineUserModule) Pub() bool     { return false }
func (*OnlineUserModule) Param() string { return "id" }
