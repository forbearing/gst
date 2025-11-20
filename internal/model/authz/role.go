package modelauthz

import (
	"errors"
	"strings"

	"github.com/forbearing/gst/authz/rbac"
	"github.com/forbearing/gst/database"
	"github.com/forbearing/gst/model"
	"github.com/forbearing/gst/types"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"gorm.io/datatypes"
)

type Role struct {
	Name    string `json:"name,omitempty" schema:"name" gorm:"size:191;unique"`
	Code    string `json:"code,omitempty" schema:"code" gorm:"size:191;unique"`
	Default *bool  `json:"default,omitempty" schema:"default"` // default role

	// Menu 相关字段指定了该角色拥有哪些菜单
	MenuIds        datatypes.JSONSlice[string] `json:"menu_ids,omitempty"`
	MenuPartialIds datatypes.JSONSlice[string] `json:"menu_partial_ids,omitempty"` // 部分选中的角色菜单, 父节点下面的子节点被选中了, 但是没有全部选中.
	ButtonIds      datatypes.JSONSlice[string] `json:"button_ids,omitempty"`       // 角色拥有的按钮权限

	Menus        []*Menu `json:"menus,omitempty" gorm:"-"` // 角色菜单
	MenuPartials []*Menu `json:"menu_partials,omitempty" gorm:"-"`

	model.Base
}

func (r *Role) Purge() bool                                { return true }
func (r *Role) CreateBefore(ctx *types.ModelContext) error { return r.validate(ctx) }

// CreateAfter will creates the role's permissions.
func (r *Role) CreateAfter(ctx *types.ModelContext) error {
	if err := database.Database[*Role](ctx.DatabaseContext()).Get(r, r.ID); err != nil {
		return err
	}
	e1 := r.UpdatePermission(ctx)
	e2 := rbac.RBAC().AddRole(r.Code)
	return errors.Join(e1, e2)
}

// UpdateBefore will delete the old role's permissions and create the new role's permissions.
// more details see "UpdatePermission".
func (r *Role) UpdateBefore(ctx *types.ModelContext) error {
	e1 := r.UpdatePermission(ctx)
	e2 := rbac.RBAC().AddRole(r.Code)
	return errors.Join(e1, e2)
}

// DeleteBefore will delete the role's permissions
func (r *Role) DeleteBefore(ctx *types.ModelContext) error {
	// The delete request always don't have role id, so we should get the role from database.
	if err := database.Database[*Role](ctx.DatabaseContext()).Get(r, r.ID); err != nil {
		return err
	}

	if err := rbac.RBAC().RemoveRole(r.Code); err != nil {
		return err
	}

	// removes the role's permissions
	menus := make([]*Menu, 0)
	permissions := make([]*Permission, 0)
	if err := database.Database[*Menu](ctx.DatabaseContext()).
		WithQuery(&Menu{Base: model.Base{ID: strings.Join(r.MenuIds, ",")}}).
		List(&menus); err != nil {
		return err
	}
	for _, m := range menus {
		result := make([]*Permission, 0)
		// query multiple permissions
		if err := database.Database[*Permission](ctx.DatabaseContext()).
			WithQuery(&Permission{Resource: strings.Join(m.API, ",")}).
			List(&result); err != nil {
			zap.S().Error(err)
			return err
		}
		permissions = append(permissions, result...)
	}

	// revoke the role's permissions
	for _, p := range permissions {
		if err := rbac.RBAC().RevokePermission(r.Code, p.Resource, p.Action); err != nil {
			return err
		}
	}
	return nil
}

// UpdatePermission must run in the "UpdateBefore" hook.
// It uses a brute-force strategy: revoke all existing policies for the role,
// then grant permissions derived from the current menus. This avoids any
// unknown leftovers in the casbin_rule table and ensures strong consistency.
func (r *Role) UpdatePermission(ctx *types.ModelContext) error {
	// We should always iterate role's "MenuIds", not "MenuPartialIds".
	// "MenuIds" is the frontend menus, "MenuPartialIds" is the frontend menus group that has no menus.
	// A "Menu" contains one or multiple backend apis, each api binding one or multiple permissions.

	var (
		newMenus       = make([]*Menu, 0)
		newPermissions = make([]*Permission, 0)
	)

	o := new(Role)
	if err := database.Database[*Role](ctx.DatabaseContext()).Get(o, r.ID); err != nil {
		zap.S().Error(err)
		return err
	}

	// query the new role's menus
	if err := database.Database[*Menu](ctx.DatabaseContext()).
		WithQuery(&Menu{Base: model.Base{ID: strings.Join(r.MenuIds, ",")}}).
		List(&newMenus); err != nil {
		zap.S().Error(err)
		return err
	}

	// query the new role's permissions
	for _, m := range newMenus {
		// zap.S().Infow("menu", "label", m.Label, "api", m.API)
		result := make([]*Permission, 0)
		if err := database.Database[*Permission](ctx.DatabaseContext()).
			// query the menu's permissions, multiple resources seperated by ","
			WithQuery(&Permission{Resource: strings.Join(m.API, ",")}).
			List(&result); err != nil {
			zap.S().Error(err)
			return err
		}
		newPermissions = append(newPermissions, result...)
	}

	for _, p := range newPermissions {
		zap.S().Infow("new permission", "role", r.Code, "resource", p.Resource, "action", p.Action, "effect", EffectAllow)
	}

	// revoke all existing policies for this role to avoid leftovers
	if err := rbac.RBAC().RevokePermission(r.Code, "", ""); err != nil {
		zap.S().Error(err)
		return err
	}
	// grant the new role's permissions
	for _, p := range newPermissions {
		if err := rbac.RBAC().GrantPermission(r.Code, p.Resource, p.Action); err != nil {
			zap.S().Error(err)
			return err
		}
	}

	zap.S().Infow("update role", "old", o.Code, "new", r.Code)

	return nil
}

// validate will validate the role's name and code and ensure the role not exists.
func (r *Role) validate(ctx *types.ModelContext) error {
	r.Name = strings.TrimSpace(r.Name)
	r.Code = strings.TrimSpace(r.Code)
	if len(r.Name) == 0 {
		return errors.New("role name is required")
	}
	if len(r.Code) == 0 {
		return errors.New("role code is required")
	}
	// check the role whether exists.
	roles := make([]*Role, 0)
	if err := database.Database[*Role](ctx.DatabaseContext()).
		WithLimit(1).
		WithQuery(&Role{Name: r.Name, Code: r.Code}).
		List(&roles); err != nil {
		return err
	}
	if len(roles) > 0 {
		return errors.New("role already exists")
	}

	// role "code" support changed, so the generated hash ID cannot be used.
	{
		// // Ensure the role with the same name/code share the same ID.
		// // If the role already exists, set same id to just update it.
		// r.SetID(util.HashID(r.Code))
	}

	return nil
}

func (r *Role) MarshalLogObject(enc zapcore.ObjectEncoder) error {
	if r == nil {
		return nil
	}
	enc.AddString("code", r.Code)
	enc.AddString("name", r.Name)
	_ = enc.AddObject("base", &r.Base)
	return nil
}
