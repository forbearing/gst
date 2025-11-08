package authz

import (
	"fmt"

	"github.com/cockroachdb/errors"
	"github.com/forbearing/gst/authz/rbac"
	"github.com/forbearing/gst/database"
	"github.com/forbearing/gst/model"
	"github.com/forbearing/gst/service"
	"github.com/forbearing/gst/types"
	"github.com/forbearing/gst/types/consts"
	"github.com/forbearing/gst/util"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

func init() {
	model.Register[*UserRole]()
}

type UserRole struct {
	UserID string `json:"user_id,omitempty" schema:"user_id"`
	RoleID string `json:"role_id,omitempty" schema:"role_id"`

	User string `json:"user,omitempty" schema:"user"` // 用户名,只是为了方便查询
	Role string `json:"role,omitempty" schema:"role"` // 角色名,只是为了方便查询

	model.Base
}

func (*UserRole) Purge() bool { return true }

func (r *UserRole) CreateBefore(ctx *types.ModelContext) error {
	if len(r.UserID) == 0 {
		return errors.New("user_id is required")
	}
	if len(r.RoleID) == 0 {
		return errors.New("role_id is required")
	}
	// expands field: user and role
	user, role := new(model.User), new(Role)
	if err := database.Database[*model.User](ctx.DatabaseContext()).Get(user, r.UserID); err != nil {
		return err
	}
	if err := database.Database[*Role](ctx.DatabaseContext()).Get(role, r.RoleID); err != nil {
		return err
	}
	r.User, r.Role = user.Name, role.Name

	// If the user already has the role, set same id to just update it.
	r.SetID(util.HashID(r.UserID, r.RoleID))

	return nil
}

func (r *UserRole) CreateAfter(ctx *types.ModelContext) error {
	if err := database.Database[*UserRole](ctx.DatabaseContext()).Update(r); err != nil {
		return err
	}
	// NOTE: must be role name not role id.
	if err := rbac.RBAC().AssignRole(r.UserID, r.Role); err != nil {
		return err
	}

	// update casbin_rule field: `user`, `role`, `remark`
	user := new(model.User)
	if err := database.Database[*model.User](ctx.DatabaseContext()).Get(user, r.UserID); err != nil {
		return err
	}
	casbinRules := make([]*CasbinRule, 0)
	if err := database.Database[*CasbinRule](ctx.DatabaseContext()).WithLimit(1).WithQuery(&CasbinRule{V0: r.UserID, V1: r.Role}).List(&casbinRules); err != nil {
		return err
	}
	if len(casbinRules) > 0 {
		casbinRules[0].User = user.Name
		casbinRules[0].Role = r.Role
		casbinRules[0].Remark = util.ValueOf(fmt.Sprintf("%s -> %s", r.User, r.Role))
		return database.Database[*CasbinRule](ctx.DatabaseContext()).Update(casbinRules[0])
	}
	return nil
}

func (r *UserRole) DeleteBefore(ctx *types.ModelContext) error {
	// The delete request always don't have user_id and role_id, so we should get the role from database.
	if err := database.Database[*UserRole](ctx.DatabaseContext()).Get(r, r.ID); err != nil {
		return err
	}
	// NOTE: must be role name not role id.
	return rbac.RBAC().UnassignRole(r.UserID, r.Role)
}

func (r *UserRole) MarshalLogObject(enc zapcore.ObjectEncoder) error {
	if r == nil {
		return nil
	}
	enc.AddString("user_id", r.UserID)
	enc.AddString("role_id", r.RoleID)
	enc.AddString("user", r.User)
	enc.AddString("role", r.Role)
	_ = enc.AddObject("base", &r.Base)
	return nil
}

type UserRoleService struct {
	service.Base[*UserRole, *UserRole, *UserRole]
}

// DeleteAfter support filter and delete multiple user_roles by query parameter `user` and `role`.
func (r *UserRoleService) DeleteAfter(ctx *types.ServiceContext, userRole *UserRole) error {
	log := r.WithServiceContext(ctx, consts.PHASE_DELETE_AFTER)
	user := ctx.URL.Query().Get("user")
	role := ctx.URL.Query().Get("role")

	userRoles := make([]*UserRole, 0)
	if err := database.Database[*UserRole](ctx.DatabaseContext()).WithQuery(&UserRole{User: user, Role: role}).List(&userRoles); err != nil {
		log.Error(err)
		return err
	}
	for _, rb := range userRoles {
		log.Infoz("will delete user role", zap.Object("user_role", rb))
	}
	if err := database.Database[*UserRole](ctx.DatabaseContext()).WithPurge().Delete(userRoles...); err != nil {
		return err
	}

	return nil
}
