package authz

import (
	"errors"
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
	"go.uber.org/zap"
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
	// 本地账号 root 是超级管理员, 不对其进行过滤
	if ctx.Username == "root" || ctx.Username == "admin" {
		return nil
	}

	user := new(User)
	if err := database.Database[*User](ctx.DatabaseContext()).Get(user, ctx.UserID); err != nil {
		log.Error(err)
		return err
	}

	// 通过 UserRole 来找到 role
	userRole := make([]*UserRole, 0)
	if err := database.Database[*UserRole](ctx.DatabaseContext()).WithQuery(&UserRole{UserID: ctx.UserID}).List(&userRole); err != nil {
		log.Error(err)
		return err
	}

	var roleID string
	if len(userRole) > 0 {
		if len(userRole[0].RoleID) > 0 {
			roleID = userRole[0].RoleID
		}
	}
	if len(userRole) > 0 {
		log.Infoz("", zap.Object("userRole", userRole[0]))
	}

	// 如果通过 rolebinding 没找到 role, 则使用默认的 role
	if len(roleID) == 0 {
		roles := make([]*Role, 0)
		if err := database.Database[*Role](ctx.DatabaseContext()).WithQuery(&Role{Default: util.ValueOf(true)}).WithLimit(1).List(&roles); err != nil {
			log.Error(err)
			return err
		}
		if len(roles) == 0 {
			log.Error("not found default role")
			return errors.New("not found default role")
		}
		if len(roles[0].ID) == 0 {
			log.Error("not found default role")
			return errors.New("not found default role")
		}
		roleID = roles[0].ID
		log.Infoz("", zap.Object("role", roles[0]))
	}

	// fmt.Println("----- rolebindings", rolebindings)

	// 获取多个 role
	// 找出所有 rolebindings 中的 role.
	roleIds := make([]string, 0)
	for _, ur := range userRole {
		roleIds = append(roleIds, ur.RoleID)
	}
	// Ensure the resolved roleID is included when user has no explicit role binding.
	// This guarantees default role is applied to menu filtering.
	if len(roleID) > 0 && !lo.Contains(roleIds, roleID) {
		roleIds = append(roleIds, roleID)
	}
	// fmt.Println("----- roleIds", roleIds)
	roles := make([]*Role, 0)
	if err := database.Database[*Role](ctx.DatabaseContext()).WithQuery(&Role{Base: model.Base{ID: strings.Join(roleIds, ",")}}).List(&roles); err != nil {
		log.Error(err)
		return err
	}
	// fmt.Println("---- roles", roles)

	// 只获取一个 role
	// role := new(Role)
	// if err := database.Database[*Role]().Get(role, roleID); err != nil {
	// 	log.Error(err)
	// 	return err
	// }
	// if len(role.ID) == 0 {
	// 	log.Error("not found role")
	// 	return errors.New("not found role")
	// }

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

// // 递归过滤出当前角色所拥有的菜单. 作用于 menu.Children 字段.
// func filter(menu *Menu, menuMap map[string]struct{}) {
// 	if len(menu.Children) > 0 {
// 		menu.Children = lo.Filter[*Menu](menu.Children, func(item *Menu, _ int) bool {
// 			if _, ok := menuMap[item.ID]; ok {
// 				return true
// 			} else {
// 				return false
// 			}
// 		})
// 		for i := range menu.Children {
// 			filter(menu.Children[i], menuMap)
// 		}
// 	}
// }
