package authz

import (
	"github.com/forbearing/gst/database"
	modelauthz "github.com/forbearing/gst/internal/model/authz"
	"github.com/forbearing/gst/service"
	"github.com/forbearing/gst/types"
	"github.com/forbearing/gst/types/consts"
	"go.uber.org/zap"
)

type RoleModule struct{}

func (*RoleModule) Service() types.Service[*Role, *Role, *Role] {
	return &RoleService{}
}
func (*RoleModule) Route() string { return "roles" }
func (*RoleModule) Pub() bool     { return false }
func (*RoleModule) Param() string { return "id" }

type Role = modelauthz.Role

type RoleService struct {
	service.Base[*Role, *Role, *Role]
}

// DeleteAfter support filter and delete multiple roles by query parameter `name`.
func (r *RoleService) DeleteAfter(ctx *types.ServiceContext, role *Role) error {
	log := r.WithServiceContext(ctx, consts.PHASE_DELETE_AFTER)
	name := ctx.URL.Query().Get("name")
	if len(name) == 0 {
		return nil
	}

	roles := make([]*Role, 0)
	if err := database.Database[*Role](ctx.DatabaseContext()).WithQuery(&Role{Name: name}).List(&roles); err != nil {
		log.Error(err)
		return err
	}
	for _, role := range roles {
		log.Infoz("will delete role", zap.Object("role", role))
	}
	if err := database.Database[*Role](ctx.DatabaseContext()).WithPurge().Delete(roles...); err != nil {
		log.Error(err)
		return err
	}

	return nil
}
