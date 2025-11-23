package iam

import (
	"github.com/forbearing/gst/module"
	"github.com/forbearing/gst/types/consts"
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
//   - CREATE /api/iam/groups
//   - DELETE /api/iam/groups/:id
//   - POST   /api/iam/groups
//   - PUT    /api/iam/groups/:id
//   - PATCH  /api/iam/groups/:id
//   - GET    /api/iam/groups
//   - GET    /api/iam/groups/:id
//   - POST   /api/heartbeat
//   - POST   /api/iam/login
//   - POST   /api/iam/logout
//   - GET    /api/iam/me
//   - GET    /api/online-users
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
		*MeRsp,
		*MeService](
		&MeModule{},
		consts.PHASE_GET,
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
}
