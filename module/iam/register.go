package iam

import (
	modeliam "github.com/forbearing/gst/internal/model/iam"
	serviceiam "github.com/forbearing/gst/internal/service/iam"
	"github.com/forbearing/gst/module"
	"github.com/forbearing/gst/types/consts"
)

const SessionNamespace = modeliam.SessionNamespace

var (
	SessionRedisKey = serviceiam.SessionRedisKey
	SessionID       = serviceiam.SessionID
)

type Config struct {
	EnableTenant bool // default disable tenant.
}

// Register registers iam modules,
//
// Models:
//   - ChangePassword
//   - Group
//
// Routes:
//   - POST   /api/iam/change-password
//   - POST   /api/iam/groups
//   - DELETE /api/iam/groups/:id
//   - PUT    /api/iam/groups/:id
//   - PATCH  /api/iam/groups/:id
//   - GET    /api/iam/groups
//   - GET    /api/iam/groups/:id
//   - POST   /api/heartbeat
//   - POST   /api/iam/login
//   - POST   /api/iam/logout
//   - POST   /api/offline
//   - GET    /api/me
//   - GET    /api/online-users
//   - POST   /api/iam/signup
//   - POST   /api/iam/tenants
//   - DELETE /api/iam/tenants/:id
//   - PUT    /api/iam/tenants/:id
//   - PATCH  /api/iam/tenants/:id
//   - GET    /api/iam/tenants
//   - GET    /api/iam/tenants/:id
//   - POST   /api/iam/users
//   - DELETE /api/iam/users/:id
//   - PUT    /api/iam/users/:id
//   - PATCH  /api/iam/users/:id
//   - GET    /api/iam/users
//   - GET    /api/iam/users/:id
func Register(...Config) {
	// Use module "ChangePasswordModule"
	module.Use[
		*ChangePassword,
		*ChangePasswordReq,
		*ChangePasswordRsp,
		*ChangePasswordService](
		&ChangePasswordModule{},
		consts.PHASE_CREATE,
	)

	// Use module "GroupModule"
	module.Use[
		*Group,
		*Group,
		*Group,
		*GroupService](
		&GroupModule{},
		consts.PHASE_CREATE,
		consts.PHASE_DELETE,
		consts.PHASE_UPDATE,
		consts.PHASE_PATCH,
		consts.PHASE_LIST,
		consts.PHASE_GET,
	)

	// Use module "HeartbeatModule"
	module.Use[
		*Heartbeat,
		*Heartbeat,
		*Heartbeat,
		*HeartbeatService](
		&HeartbeatModule{},
		consts.PHASE_CREATE,
	)

	// Use module "LoginModule"
	module.Use[
		*Login,
		*LoginReq,
		*LoginRsp,
		*LoginService](
		&LoginModule{},
		consts.PHASE_CREATE,
	)

	// Use module "LogoutModule"
	module.Use[
		*Logout,
		*Logout,
		*LogoutRsp,
		*LogoutService](
		&LogoutModule{},
		consts.PHASE_CREATE,
	)

	// Use module "MeModule"
	module.Use[
		*Me,
		*Me,
		MeRsp,
		*MeService](
		&MeModule{},
		consts.PHASE_GET,
	)

	// Use module "OfflineModule"
	module.Use[
		*Offline,
		*OfflineReq,
		*Offline,
		*OfflineService](
		&OfflineModule{},
		consts.PHASE_CREATE,
	)

	// Use module "OnlineUserModule"
	module.Use[
		*OnlineUser,
		*OnlineUser,
		*OnlineUser,
		*OnlineUserService](
		&OnlineUserModule{},
		consts.PHASE_LIST,
	)

	// Use module "SignupModule"
	module.Use[
		*Signup,
		*SignupReq,
		*SignupRsp,
		*SignupService](
		&SignupModule{},
		consts.PHASE_CREATE,
	)

	// Use module "TenantModule"
	module.Use[
		*Tenant,
		*Tenant,
		*Tenant,
		*TenantService](
		&TenantModule{},
		consts.PHASE_CREATE,
		consts.PHASE_DELETE,
		consts.PHASE_UPDATE,
		consts.PHASE_PATCH,
		consts.PHASE_LIST,
		consts.PHASE_GET,
	)

	// Use module "UserModule"
	module.Use[
		*User,
		*User,
		*User,
		*UserService](
		&UserModule{},
		consts.PHASE_CREATE,
		consts.PHASE_DELETE,
		consts.PHASE_UPDATE,
		consts.PHASE_PATCH,
		consts.PHASE_LIST,
		consts.PHASE_GET,
	)
}
