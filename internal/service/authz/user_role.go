package serviceauthz

import (
	"github.com/forbearing/gst/database"
	modelauthz "github.com/forbearing/gst/internal/model/authz"
	"github.com/forbearing/gst/service"
	"github.com/forbearing/gst/types"
	"github.com/forbearing/gst/types/consts"
	"go.uber.org/zap"
)

type UserRoleService struct {
	service.Base[*modelauthz.UserRole, *modelauthz.UserRole, *modelauthz.UserRole]
}

// DeleteAfter support filter and delete multiple user_roles by query parameter `user` and `role`.
func (s *UserRoleService) DeleteAfter(ctx *types.ServiceContext, userRole *modelauthz.UserRole) error {
	log := s.WithServiceContext(ctx, consts.PHASE_DELETE_AFTER)
	user := ctx.URL.Query().Get("user")
	role := ctx.URL.Query().Get("role")

	userRoles := make([]*modelauthz.UserRole, 0)
	if err := database.Database[*modelauthz.UserRole](ctx.DatabaseContext()).WithQuery(&modelauthz.UserRole{User: user, Role: role}).List(&userRoles); err != nil {
		log.Error(err)
		return err
	}
	for _, rb := range userRoles {
		log.Infoz("will delete user role", zap.Object("user_role", rb))
	}
	if err := database.Database[*modelauthz.UserRole](ctx.DatabaseContext()).WithPurge().Delete(userRoles...); err != nil {
		return err
	}

	return nil
}
