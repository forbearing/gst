package iam

import (
	"time"

	"github.com/forbearing/gst/cronjob"
	cronjobiam "github.com/forbearing/gst/internal/cronjob/iam"
	modeliam "github.com/forbearing/gst/internal/model/iam"
	serviceiam "github.com/forbearing/gst/internal/service/iam"
	"github.com/forbearing/gst/middleware"
	"github.com/forbearing/gst/model"
	"github.com/forbearing/gst/module"
	"github.com/forbearing/gst/types/consts"
)

type (
	Session = modeliam.Session
	Token   = modeliam.Token
)

// iamConfig stores the configuration for iam module
var iamConfig Config

// Config is the configuration for iam module.
type Config struct {
	EnableTenant      bool          // EnableTenant enables tenant module, default is false
	DefaultUsers      []*User       // DefaultUsers are default users to create on registration
	SessionExpiration time.Duration // SessionExpiration is the session expiration time, default is 8 hours
}

// Register registers iam modules,
//
// Models:
//   - ChangePassword
//   - Group
//   - Heartbeat
//   - Login
//   - Logout
//   - Offline
//   - Me
//   - OnlineUser
//   - Signup
//   - Tenant
//   - User
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
//   - POST   /api/login
//   - POST   /api/logout
//   - POST   /api/offline
//   - GET    /api/me
//   - GET    /api/online-users
//   - POST   /api/signup
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
//
// Middleware:
//   - IAMSession
//
// Default disable Tenant module, use `EnableTenant` to enable it.
// Default session expiration time is 8 hours, use `SessionExpiration` to customize it.
//
// NOTE: iam modules register must before "authz" modules register.
// because "authz" registered middleware "Authz" depend on iam modules registered middleware "IAMSession".
func Register(config ...Config) {
	cfg := Config{
		SessionExpiration: 8 * time.Hour, // default session expiration time
	}
	if len(config) > 0 {
		cfg = config[0]
		// Set default session expiration if not provided
		if cfg.SessionExpiration == 0 {
			cfg.SessionExpiration = 8 * time.Hour
		}
	}

	// Store config globally
	iamConfig = cfg

	// Set session expiration in service layer
	serviceiam.SetSessionExpiration(cfg.SessionExpiration)

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
		consts.PHASE_LIST,
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

	if cfg.EnableTenant {
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
	}

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
	// create default users
	if len(cfg.DefaultUsers) > 0 {
		for _, u := range cfg.DefaultUsers {
			if err := modeliam.GenerateHashedPassword(u); err != nil {
				panic(err)
			}
		}
		model.Register(cfg.DefaultUsers...)
	}

	middleware.RegisterAuth(middleware.IAMSession())

	// cleanup the oneline user that not active every 30 seconds, will run immediately after application bootstrap.
	cronjob.Register(cronjobiam.CleanupOnlineUser, "*/30 * * * * *", "cleanup online user", cronjob.Config{RunImmediately: true})
}

// GetSessionExpiration returns the configured session expiration time.
// If not configured, it returns the default value of 8 hours.
func GetSessionExpiration() time.Duration {
	if iamConfig.SessionExpiration == 0 {
		return 8 * time.Hour
	}
	return iamConfig.SessionExpiration
}
