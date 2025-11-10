package authz

import (
	"github.com/forbearing/gst/module"
	"github.com/forbearing/gst/service"
	"github.com/forbearing/gst/types/consts"
)

// Register register modules: Permission, Role, RolePermission, UserRole.
//
// Modules:
//   - Permission
//   - Role
//   - RolePermission
//   - UserRole
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
}
