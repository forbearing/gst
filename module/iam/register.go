package iam

import (
	"time"

	"github.com/forbearing/gst/cronjob"
	cronjobiam "github.com/forbearing/gst/internal/cronjob/iam"
	modeliam "github.com/forbearing/gst/internal/model/iam"
	serviceiam "github.com/forbearing/gst/internal/service/iam"
	serviceiamaccount "github.com/forbearing/gst/internal/service/iam/account"
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

// Register registers IAM models, API routes, middleware, and scheduled jobs.
//
// Models:
//   - ChangePassword
//   - ResetPassword
//   - AccountStatus
//   - EmailChangeConfirm
//   - EmailChangeCancel
//   - EmailChangeRequest
//   - EmailChangeResend
//   - EmailPasswordResetConfirm
//   - EmailPasswordResetRequest
//   - EmailVerificationConfirm
//   - EmailVerificationRequest
//   - EmailVerificationResend
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
// API Routes:
//
// Public auth routes:
//   - POST   /api/login
//   - POST   /api/signup
//
// Session routes:
//   - POST   /api/logout
//   - POST   /api/heartbeat
//   - GET    /api/me
//   - POST   /api/offline
//   - GET    /api/online-users
//
// Account management routes:
//   - POST   /api/iam/change-password
//   - POST   /api/iam/reset-password
//   - POST   /api/iam/account-status
//
// IAM resource routes:
//   - POST   /api/iam/users
//   - DELETE /api/iam/users/:id
//   - PUT    /api/iam/users/:id
//   - PATCH  /api/iam/users/:id
//   - GET    /api/iam/users
//   - GET    /api/iam/users/:id
//   - POST   /api/iam/groups
//   - DELETE /api/iam/groups/:id
//   - PUT    /api/iam/groups/:id
//   - PATCH  /api/iam/groups/:id
//   - GET    /api/iam/groups
//   - GET    /api/iam/groups/:id
//   - POST   /api/iam/tenants
//   - DELETE /api/iam/tenants/:id
//   - PUT    /api/iam/tenants/:id
//   - PATCH  /api/iam/tenants/:id
//   - GET    /api/iam/tenants
//   - GET    /api/iam/tenants/:id
//
// Email workflow routes:
//   - POST   /api/iam/email/change-confirm
//   - POST   /api/iam/email/change-cancel
//   - POST   /api/iam/email/change-request
//   - POST   /api/iam/email/change-resend
//   - POST   /api/iam/email/password-reset-confirm
//   - POST   /api/iam/email/password-reset-request
//   - POST   /api/iam/email/verification-confirm
//   - POST   /api/iam/email/verification-request
//   - POST   /api/iam/email/verification-resend
//
// Middleware:
//   - IAMSession for protected IAM routes and session-aware APIs
//
// Scheduled jobs:
//   - CleanupOnlineUser runs every 30 seconds and starts immediately after bootstrap
//
// Configuration:
//   - Tenant routes are registered only when EnableTenant is true
//   - SessionExpiration defaults to 8 hours when not configured
//
// NOTE: Register IAM modules before authz modules because authz middleware depends on IAMSession.
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

	module.Use(module.NewWrapper("/login", "id", true, &serviceiamaccount.LoginService{}), consts.PHASE_CREATE)
	module.Use(module.NewWrapper("/logout", "id", false, &serviceiamaccount.LogoutService{}), consts.PHASE_CREATE)
	module.Use(module.NewWrapper("/signup", "id", true, &serviceiamaccount.SignupService{}), consts.PHASE_CREATE)
	module.Use(module.NewWrapper("/iam/change-password", "id", false, &serviceiamaccount.ChangePasswordService{}), consts.PHASE_CREATE)
	module.Use(module.NewWrapper("/iam/reset-password", "id", false, &serviceiamaccount.ResetPasswordService{}), consts.PHASE_CREATE)
	module.Use(module.NewWrapper("/iam/account-status", "id", false, &serviceiamaccount.AccountStatusService{}), consts.PHASE_CREATE)
	module.Use(module.NewWrapper("/heartbeat", "id", false, &serviceiam.HeartbeatService{}), consts.PHASE_CREATE)
	module.Use(module.NewWrapper("/me", "id", false, &serviceiam.MeService{}), consts.PHASE_LIST)
	module.Use(module.NewWrapper("/offline", "id", false, &serviceiam.OfflineService{}), consts.PHASE_CREATE)
	module.Use(
		module.NewWrapper[*User, *User, *User]("/iam/users", "id", false, &serviceiam.UserService{}),
		consts.PHASE_CREATE,
		consts.PHASE_DELETE,
		consts.PHASE_UPDATE,
		consts.PHASE_PATCH,
		consts.PHASE_LIST,
		consts.PHASE_GET,
	)
	module.Use(
		module.NewWrapper[*Group, *Group, *Group]("/iam/groups", "id", false),
		consts.PHASE_CREATE,
		consts.PHASE_DELETE,
		consts.PHASE_UPDATE,
		consts.PHASE_PATCH,
		consts.PHASE_LIST,
		consts.PHASE_GET,
	)
	if cfg.EnableTenant {
		module.Use(
			module.NewWrapper[*Tenant, *Tenant, *Tenant]("/iam/tenants", "id", false),
			consts.PHASE_CREATE,
			consts.PHASE_DELETE,
			consts.PHASE_UPDATE,
			consts.PHASE_PATCH,
			consts.PHASE_LIST,
			consts.PHASE_GET,
		)
	}
	module.Use(
		module.NewWrapper[*OnlineUser, *OnlineUser, *OnlineUser]("/online-users", "id", false),
		consts.PHASE_LIST,
	)

	// Use module "EmailChangeConfirmModule"
	module.Use[
		*EmailChangeConfirm,
		*EmailChangeConfirmReq,
		*EmailChangeConfirmRsp](
		&EmailChangeConfirmModule{},
		consts.PHASE_CREATE,
	)

	// Use module "EmailChangeCancelModule"
	module.Use[
		*EmailChangeCancel,
		*EmailChangeCancelReq,
		*EmailChangeCancelRsp](
		&EmailChangeCancelModule{},
		consts.PHASE_CREATE,
	)

	// Use module "EmailChangeRequestModule"
	module.Use[
		*EmailChangeRequest,
		*EmailChangeRequestReq,
		*EmailChangeRequestRsp](
		&EmailChangeRequestModule{},
		consts.PHASE_CREATE,
	)

	// Use module "EmailChangeResendModule"
	module.Use[
		*EmailChangeResend,
		*EmailChangeResendReq,
		*EmailChangeResendRsp](
		&EmailChangeResendModule{},
		consts.PHASE_CREATE,
	)

	// Use module "EmailPasswordResetConfirmModule"
	module.Use[
		*EmailPasswordResetConfirm,
		*EmailPasswordResetConfirmReq,
		*EmailPasswordResetConfirmRsp](
		&EmailPasswordResetConfirmModule{},
		consts.PHASE_CREATE,
	)

	// Use module "EmailPasswordResetRequestModule"
	module.Use[
		*EmailPasswordResetRequest,
		*EmailPasswordResetRequestReq,
		*EmailPasswordResetRequestRsp](
		&EmailPasswordResetRequestModule{},
		consts.PHASE_CREATE,
	)

	// Use module "EmailVerificationConfirmModule"
	module.Use[
		*EmailVerificationConfirm,
		*EmailVerificationConfirmReq,
		*EmailVerificationConfirmRsp](
		&EmailVerificationConfirmModule{},
		consts.PHASE_CREATE,
	)

	// Use module "EmailVerificationRequestModule"
	module.Use[
		*EmailVerificationRequest,
		*EmailVerificationRequestReq,
		*EmailVerificationRequestRsp](
		&EmailVerificationRequestModule{},
		consts.PHASE_CREATE,
	)

	// Use module "EmailVerificationResendModule"
	module.Use[
		*EmailVerificationResend,
		*EmailVerificationResendReq,
		*EmailVerificationResendRsp](
		&EmailVerificationResendModule{},
		consts.PHASE_CREATE,
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
