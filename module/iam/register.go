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
}
