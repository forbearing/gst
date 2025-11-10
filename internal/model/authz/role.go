package modelauthz

import (
	"errors"
	"strings"

	"github.com/forbearing/gst/authz/rbac"
	"github.com/forbearing/gst/database"
	"github.com/forbearing/gst/model"
	"github.com/forbearing/gst/types"
	"github.com/forbearing/gst/util"
	"go.uber.org/zap/zapcore"
	"gorm.io/datatypes"
)

type Role struct {
	Name    string `json:"name,omitempty" schema:"name"`
	Code    string `json:"code,omitempty" schema:"code"`
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
func (r *Role) CreateAfter(ctx *types.ModelContext) error  { return rbac.RBAC().AddRole(r.Code) }
func (r *Role) UpdateBefore(ctx *types.ModelContext) error { return rbac.RBAC().AddRole(r.Code) }
func (r *Role) DeleteBefore(ctx *types.ModelContext) error {
	// The delete request always don't have role id, so we should get the role from database.
	if err := database.Database[*Role](ctx.DatabaseContext()).Get(r, r.ID); err != nil {
		return err
	}
	if len(r.Code) > 0 {
		return rbac.RBAC().RemoveRole(r.Code)
	}
	return nil
}

func (r *Role) validate(ctx *types.ModelContext) error {
	r.Name = strings.TrimSpace(r.Name)
	r.Code = strings.TrimSpace(r.Code)
	if len(r.Name) == 0 {
		return errors.New("name is required")
	}
	if len(r.Code) == 0 {
		return errors.New("code is required")
	}
	// Ensure the role with the same name/code share the same ID.
	// If the role already exists, set same id to just update it.
	r.SetID(util.HashID(r.Code))

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
