package authz

import (
	"os"

	"github.com/forbearing/gst/config"
	"github.com/forbearing/gst/model"
	"github.com/forbearing/gst/module"
	"github.com/forbearing/gst/service"
	"github.com/forbearing/gst/types/consts"
)

func init() {
	// Enable RBAC
	os.Setenv(config.AUTH_RBAC_ENABLE, "true")
}

// Register register modules: Permission, Role, RolePermission, UserRole.
//
// Modules:
//   - Permission
//   - Role
//   - RolePermission
//   - UserRole
//   - CasbinRule
//   - Menu
//
// Routes:
//   - GET    /api/authz/permissions
//   - GET    /api/authz/permissions/:id
//   - POST   /api/authz/roles
//   - DELETE /api/authz/roles/:id
//   - PUT    /api/authz/roles/:id
//   - PATCH  /api/authz/roles/:id
//   - GET    /api/authz/roles
//   - GET    /api/authz/roles/:id
//   - POST   /api/authz/role-permissions
//   - DELETE /api/authz/role-permissions/:id
//   - PUT    /api/authz/role-permissions/:id
//   - PATCH  /api/authz/role-permissions/:id
//   - GET    /api/authz/role-permissions
//   - GET    /api/authz/role-permissions/:id
//   - POST   /api/authz/user-roles
//   - DELETE /api/authz/user-roles/:id
//   - PUT    /api/authz/user-roles/:id
//   - PATCH  /api/authz/user-roles/:id
//   - GET    /api/authz/user-roles
//   - GET    /api/authz/user-roles/:id
//   - GET    /api/menus
//   - GET    /api/apis
func Register() {
	// creates table "casbin_rule".
	model.Register[*CasbinRule]()

	// create table "menus" and creates three records.
	model.Register[*Menu](
		&Menu{Base: model.Base{ID: model.RootID}, ParentID: model.RootID},
		&Menu{Base: model.Base{ID: model.NoneID}, ParentID: model.RootID},
		&Menu{Base: model.Base{ID: model.UnknownID}, ParentID: model.RootID},
	)

	module.Use[
		*Permission,
		*Permission,
		*Permission,
		*service.Base[*Permission, *Permission, *Permission]](
		&PermissionModule{},
		consts.PHASE_LIST,
		consts.PHASE_GET,
	)

	module.Use[
		*Role,
		*Role,
		*Role,
		*RoleService](
		&RoleModule{},
		consts.PHASE_CREATE,
		consts.PHASE_DELETE,
		consts.PHASE_UPDATE,
		consts.PHASE_PATCH,
		consts.PHASE_LIST,
		consts.PHASE_GET,
	)

	module.Use[
		*RolePermission,
		*RolePermission,
		*RolePermission,
		*RolePermissionService](
		&RolePermissionModule{},
		consts.PHASE_CREATE,
		consts.PHASE_DELETE,
		consts.PHASE_UPDATE,
		consts.PHASE_PATCH,
		consts.PHASE_LIST,
		consts.PHASE_GET,
	)

	module.Use[
		*UserRole,
		*UserRole,
		*UserRole,
		*UserRoleService](
		&UserRoleModule{},
		consts.PHASE_CREATE,
		consts.PHASE_DELETE,
		consts.PHASE_UPDATE,
		consts.PHASE_PATCH,
		consts.PHASE_LIST,
		consts.PHASE_GET,
	)

	module.Use[
		*Menu,
		*Menu,
		*Menu,
		*MenuService](
		&MenuModule{},
		consts.PHASE_CREATE,
		consts.PHASE_DELETE,
		consts.PHASE_UPDATE,
		consts.PHASE_PATCH,
		consts.PHASE_LIST,
		consts.PHASE_GET,
	)

	module.Use[
		*Api,
		*Api,
		ApiRsp,
		*ApiService](
		&ApiModule{},
		consts.PHASE_LIST,
	)
}
