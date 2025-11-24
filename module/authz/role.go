package authz

import (
	modelauthz "github.com/forbearing/gst/internal/model/authz"
	serviceauthz "github.com/forbearing/gst/internal/service/authz"
	"github.com/forbearing/gst/types"
)

var _ types.Module[*Role, *Role, *Role] = (*RoleModule)(nil)

type (
	Role        = modelauthz.Role
	RoleService = serviceauthz.RoleService
	RoleModule  struct{}
)

func (*RoleModule) Service() types.Service[*Role, *Role, *Role] {
	return &RoleService{}
}
func (*RoleModule) Route() string { return "authz/roles" }
func (*RoleModule) Pub() bool     { return false }
func (*RoleModule) Param() string { return "id" }
