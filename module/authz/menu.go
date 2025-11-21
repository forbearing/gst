package authz

import (
	"reflect"
	"regexp"
	"strings"

	"github.com/forbearing/gst/database"
	internalmodel "github.com/forbearing/gst/internal/model"
	modelauthz "github.com/forbearing/gst/internal/model/authz"
	"github.com/forbearing/gst/model"
	"github.com/forbearing/gst/service"
	"github.com/forbearing/gst/types"
	"github.com/forbearing/gst/util"
	"github.com/samber/lo"
)

var _ types.Module[*Menu, *Menu, *Menu] = (*MenuModule)(nil)

type (
	Menu = modelauthz.Menu
	User = internalmodel.User
)

type MenuModule struct {
	service.Base[*Menu, *Menu, *Menu]
}

func (*MenuModule) Service() types.Service[*Menu, *Menu, *Menu] {
	return &MenuService{}
}
func (*MenuModule) Route() string { return "menus" }
func (*MenuModule) Pub() bool     { return false }
func (*MenuModule) Param() string { return "id" }

type MenuService struct {
	service.Base[*Menu, *Menu, *Menu]
}

func (m *MenuService) ListAfter(ctx *types.ServiceContext, data *[]*Menu) error {
	return m.filterByRole(ctx, data, m.WithServiceContext(ctx, ctx.GetPhase()))
}

func (m *MenuService) filterByRole(ctx *types.ServiceContext, data *[]*Menu, log types.Logger) error {
	// If Username is "root" or "admin", return directly
	if ctx.Username == "root" || ctx.Username == "admin" {
		return nil
	}

	var (
		user      = new(User)
		userRoles = make([]*UserRole, 0)
		roles     = make([]*Role, 0)
	)

	// query the current user
	if err := database.Database[*User](ctx.DatabaseContext()).Get(user, ctx.UserID); err != nil {
		log.Error(err)
		return err
	}

	// query all "UserRole" according to the current user id.
	if err := database.Database[*UserRole](ctx.DatabaseContext()).
		WithQuery(&UserRole{UserID: ctx.UserID}).
		List(&userRoles); err != nil {
		log.Error(err)
		return err
	}

	// query all "Role" according to the "UserRole"
	if len(userRoles) > 0 {
		roleIds := make([]string, 0)
		for _, ur := range userRoles {
			if len(ur.RoleID) > 0 {
				roleIds = append(roleIds, ur.RoleID)
			}
		}
		if err := database.Database[*Role](ctx.DatabaseContext()).
			WithQuery(&Role{Base: model.Base{ID: strings.Join(roleIds, ",")}}).List(&roles); err != nil {
			log.Error(err)
			return err
		}
	}
	// the user has no roles, use the default role.
	if len(roles) == 0 {
		if err := database.Database[*Role](ctx.DatabaseContext()).
			WithQuery(&Role{Default: util.ValueOf(true)}).
			List(&roles); err != nil {
			log.Error(err)
			return err
		}
	}
	if len(roles) == 0 {
		log.Warn("user has no roles and don't have default role")
		data = nil
		return nil
	}
	for _, r := range roles {
		log.Infow("role", "username", ctx.Username, "role_name", r.Name, "role_code", r.Code)
	}

	{
		menuMap := make(map[string]struct{})
		for _, role := range roles {
			for _, id := range role.MenuIds {
				menuMap[id] = struct{}{}
			}
			// 这里需要把 MenuPartialIds 加进去, 父菜单下面有多个菜单, 如果只选中了部分, 则是将 id 放在 MenuPartialIds.
			for _, id := range role.MenuPartialIds {
				menuMap[id] = struct{}{}
			}
		}
		// fmt.Println("---- menuMap", len(menuMap))

		_data := lo.Filter[*Menu](*data, func(item *Menu, _ int) bool {
			var exists, matched, ok bool
			_, exists = menuMap[item.ID]
			if exists {
				if matched, _ = regexp.MatchString(item.DomainPattern, ctx.Request.Host); matched {
					ok = true
				}
			}
			return ok
			// if _, ok := menuMap[item.ID]; ok {
			// 	return true
			// } else {
			// 	return false
			// }
		})
		for i := range _data {
			filter(ctx, _data[i], menuMap)
		}
		val := reflect.ValueOf(data)
		val.Elem().Set(reflect.ValueOf(_data))
		return nil
	}
}

// 递归过滤出当前角色所拥有的菜单. 作用于 menu.Children 字段.
func filter(ctx *types.ServiceContext, menu *Menu, menuMap map[string]struct{}) {
	if len(menu.Children) > 0 {
		menu.Children = lo.Filter[*Menu](menu.Children, func(item *Menu, _ int) bool {
			var exists, matched, ok bool
			_, exists = menuMap[item.ID]
			if exists {
				if matched, _ = regexp.MatchString(item.DomainPattern, ctx.Request.Host); matched {
					ok = true
				}
			}
			return ok
			// if _, ok := menuMap[item.ID]; ok {
			// 	return true
			// } else {
			// 	return false
			// }
		})
		for i := range menu.Children {
			filter(ctx, menu.Children[i], menuMap)
		}
	}
}
