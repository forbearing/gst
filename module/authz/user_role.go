package authz

import (
	modelauthz "github.com/forbearing/gst/internal/model/authz"
	serviceauthz "github.com/forbearing/gst/internal/service/authz"
	"github.com/forbearing/gst/types"
)

var _ types.Module[*UserRole, *UserRole, *UserRole] = (*UserRoleModule)(nil)

type (
	UserRole        = modelauthz.UserRole
	UserRoleService = serviceauthz.UserRoleService
	UserRoleModule  struct{}
)

func (*UserRoleModule) Service() types.Service[*UserRole, *UserRole, *UserRole] {
	return &UserRoleService{}
}
func (*UserRoleModule) Route() string { return "authz/user-roles" }
func (*UserRoleModule) Pub() bool     { return false }
func (*UserRoleModule) Param() string { return "id" }
