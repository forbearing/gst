package authz

import (
	"github.com/forbearing/gst/database"
	modelauthz "github.com/forbearing/gst/internal/model/authz"
	"github.com/forbearing/gst/service"
	"github.com/forbearing/gst/types"
	"github.com/forbearing/gst/types/consts"
	"go.uber.org/zap"
)

type RolePermissionModule struct{}

func (*RolePermissionModule) Service() types.Service[*RolePermission, *RolePermission, *RolePermission] {
	return &RolePermissionService{}
}

func (*RolePermissionModule) Route() string { return "role-permissions" }
func (*RolePermissionModule) Pub() bool     { return false }
func (*RolePermissionModule) Param() string { return "id" }

type RolePermission = modelauthz.RolePermission

type RolePermissionService struct {
	service.Base[*RolePermission, *RolePermission, *RolePermission]
}

// DeleteAfter support delete multiple role_permissions by query parameters `role`, `resource`, `action`
func (s *RolePermissionService) DeleteAfter(ctx *types.ServiceContext, rolePermission *modelauthz.RolePermission) error {
	log := s.WithServiceContext(ctx, consts.PHASE_DELETE_AFTER)
	role := ctx.URL.Query().Get("role")
	resource := ctx.URL.Query().Get("resource")
	action := ctx.URL.Query().Get("action")

	rolePermissions := make([]*modelauthz.RolePermission, 0)
	if err := database.Database[*modelauthz.RolePermission](ctx.DatabaseContext()).WithQuery(&modelauthz.RolePermission{
		Role:     role,
		Resource: resource,
		Action:   action,
	}).List(&rolePermissions); err != nil {
		log.Error(err)
		return err
	}
	for _, rp := range rolePermissions {
		log.Infoz("will delete role permission", zap.Object("role_permission", rp))
	}
	if err := database.Database[*modelauthz.RolePermission](ctx.DatabaseContext()).WithPurge().Delete(rolePermissions...); err != nil {
		log.Error(err)
		return err
	}

	return nil
}
