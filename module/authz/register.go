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
//   - GET    authz/permissions
//   - GET    authz/permissions/:id
//   - POST   authz/roles
//   - DELETE authz/roles/:id
//   - PUT    authz/roles/:id
//   - PATCH  authz/roles/:id
//   - GET    authz/roles
//   - GET    authz/roles/:id
//   - POST   authz/role-permissions
//   - DELETE authz/role-permissions/:id
//   - PUT    authz/role-permissions/:id
//   - PATCH  authz/role-permissions/:id
//   - GET    authz/role-permissions
//   - GET    authz/role-permissions/:id
//   - POST   authz/user-roles
//   - DELETE authz/user-roles/:id
//   - PUT    authz/user-roles/:id
//   - PATCH  authz/user-roles/:id
//   - GET    authz/user-roles
//   - GET    authz/user-roles/:id
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
}
