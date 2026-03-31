package iam

import (
	modeliam "github.com/forbearing/gst/internal/model/iam"
	serviceiam "github.com/forbearing/gst/internal/service/iam"
	"github.com/forbearing/gst/types"
)

var _ types.Module[*User, *User, *User] = (*UserModule)(nil)

type (
	User       = modeliam.User
	UserModule struct{}
)

func (*UserModule) Service() types.Service[*User, *User, *User] {
	return &serviceiam.UserService{}
}
func (*UserModule) Route() string { return "/iam/users" }
func (*UserModule) Pub() bool     { return false }
func (*UserModule) Param() string { return "id" }
