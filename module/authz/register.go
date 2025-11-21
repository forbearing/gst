package authz

import (
	"os"

	"github.com/forbearing/gst/config"
	"github.com/forbearing/gst/middleware"
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
//   - PUT    /api/authz/roles
//   - PATCH  /api/authz/roles/:id
//   - GET    /api/authz/roles
//   - GET    /api/authz/roles/:id
//   - POST   /api/authz/user-roles
//   - DELETE /api/authz/user-roles/:id
//   - PUT    /api/authz/user-roles
//   - PATCH  /api/authz/user-roles/:id
//   - GET    /api/authz/user-roles
//   - GET    /api/authz/user-roles/:id
//   - POST   /api/menus
//   - DELETE /api/menus/:id
//   - PUT    /api/menus
//   - PATCH  /api/menus/:id
//   - GET    /api/menus
//   - GET    /api/menus/:id
//   - PATCH  /api/menus/batch
//   - GET    /api/apis
//   - POST   /api/buttons
//   - DELETE /api/buttons/:id
//   - PUT    /api/buttons
//   - PATCH  /api/buttons/:id
//   - GET    /api/buttons
//   - GET    /api/buttons/:id
func Register() {
	// creates table "casbin_rule".
	model.Register[*CasbinRule]()
	middleware.Register(middleware.Authz())

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
		consts.PHASE_PATCH_MANY,
	)

	module.Use[
		*API,
		*API,
		APIRsp,
		*APIService](
		&APIModule{},
		consts.PHASE_LIST,
	)

	module.Use[
		*Button,
		*Button,
		*Button,
		*ButtonService](
		&ButtonModule{},
		consts.PHASE_CREATE,
		consts.PHASE_DELETE,
		consts.PHASE_UPDATE,
		consts.PHASE_PATCH,
		consts.PHASE_LIST,
		consts.PHASE_GET,
	)
}
