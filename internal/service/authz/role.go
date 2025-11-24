package serviceauthz

import (
	"fmt"
	"strings"

	"github.com/forbearing/gst/database"
	modelauthz "github.com/forbearing/gst/internal/model/authz"
	"github.com/forbearing/gst/service"
	"github.com/forbearing/gst/types"
	"github.com/forbearing/gst/types/consts"
	"github.com/forbearing/gst/util"
	"go.uber.org/zap"
)

type RoleService struct {
	service.Base[*modelauthz.Role, *modelauthz.Role, *modelauthz.Role]
}

// DeleteAfter support filter and delete multiple roles by query parameter `name`.
func (s *RoleService) DeleteAfter(ctx *types.ServiceContext, role *modelauthz.Role) error {
	log := s.WithServiceContext(ctx, consts.PHASE_DELETE_AFTER)
	name := ctx.URL.Query().Get("name")
	if len(name) == 0 {
		return nil
	}

	roles := make([]*modelauthz.Role, 0)
	if err := database.Database[*modelauthz.Role](ctx.DatabaseContext()).WithQuery(&modelauthz.Role{Name: name}).List(&roles); err != nil {
		log.Error(err)
		return err
	}
	for _, role := range roles {
		log.Infoz("will delete role", zap.Object("role", role))
	}
	if err := database.Database[*modelauthz.Role](ctx.DatabaseContext()).WithPurge().Delete(roles...); err != nil {
		log.Error(err)
		return err
	}

	return nil
}

func (s *RoleService) CreateAfter(ctx *types.ServiceContext, role *modelauthz.Role) error {
	return s.remarkMenus(ctx, role)
}

func (s *RoleService) UpdateAfter(ctx *types.ServiceContext, role *modelauthz.Role) error {
	return s.remarkMenus(ctx, role)
}

func (s *RoleService) PatchAfter(ctx *types.ServiceContext, role *modelauthz.Role) error {
	return s.remarkMenus(ctx, role)
}

// remarkMenus remark role about menus
func (s *RoleService) remarkMenus(ctx *types.ServiceContext, role *modelauthz.Role) error {
	log := s.WithServiceContext(ctx, ctx.GetPhase())

	menus := make([]*modelauthz.Menu, 0)
	if err := database.Database[*modelauthz.Menu](ctx.DatabaseContext()).List(&menus); err != nil {
		log.Error(err)
		return err
	}

	menuMap := make(map[string]*modelauthz.Menu)
	for _, m := range menus {
		menuMap[m.ID] = m
	}

	var sb strings.Builder
	if len(role.MenuPartialIds) > 0 {
		sb.WriteString("父菜单\n")
	}
	for _, mid := range role.MenuPartialIds {
		if menu, ok := menuMap[mid]; ok {
			sb.WriteString(fmt.Sprintf("    %s\n", menu.Label))
		}
	}
	if len(role.MenuIds) > 0 {
		sb.WriteString("\n子菜单\n")
	}
	for _, mid := range role.MenuIds {
		if menu, ok := menuMap[mid]; ok {
			sb.WriteString(fmt.Sprintf("    %s\n", menu.Label))
		}
	}

	role.Remark = util.ValueOf(strings.TrimSpace(sb.String()))

	// NOTE: Role has "UpdateBefore" hook to update role's permissions.
	// this service operations just update role's remark, so we should not invoke any "hooks" here.
	if err := database.Database[*modelauthz.Role](ctx.DatabaseContext()).WithoutHook().Update(role); err != nil {
		log.Error(err)
		return err
	}

	log.Info("update remark about menus successfully")

	return nil
}
