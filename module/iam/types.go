package iam

import (
	modeliam "github.com/forbearing/gst/internal/model/iam"
	modeliamaccount "github.com/forbearing/gst/internal/model/iam/account"
	modeliamemail "github.com/forbearing/gst/internal/model/iam/email"
	modeliamsession "github.com/forbearing/gst/internal/model/iam/session"
)

// account
type (
	LoginReq  = modeliamaccount.LoginReq
	LoginRsp  = modeliamaccount.LoginRsp
	LogoutRsp = modeliamaccount.LogoutRsp
	SignupReq = modeliamaccount.SignupReq
	SignupRsp = modeliamaccount.SignupRsp

	ChangePasswordReq = modeliamaccount.ChangePasswordReq
	ChangePasswordRsp = modeliamaccount.ChangePasswordRsp

	ResetPasswordReq = modeliamaccount.ResetPasswordReq
	ResetPasswordRsp = modeliamaccount.ResetPasswordRsp

	AccountStatusReq = modeliamaccount.AccountStatusReq
	AccountStatusRsp = modeliamaccount.AccountStatusRsp

	User   = modeliam.User
	Group  = modeliam.Group
	Tenant = modeliam.Tenant

	Session     = modeliamsession.Session
	SessionView = modeliamsession.SessionView
	Token       = modeliamsession.Token
	Heartbeat   = modeliamsession.Heartbeat
	OnlineUser  = modeliamsession.OnlineUser
	OfflineReq  = modeliamsession.OfflineReq

	CurrentListReq   = modeliamsession.CurrentListReq
	CurrentListRsp   = modeliamsession.CurrentListRsp
	CurrentDeleteReq = modeliamsession.CurrentDeleteReq
	CurrentDeleteRsp = modeliamsession.CurrentDeleteRsp

	SessionsListReq      = modeliamsession.SessionsListReq
	SessionsListRsp      = modeliamsession.SessionsListRsp
	SessionsGetReq       = modeliamsession.SessionsGetReq
	SessionsGetRsp       = modeliamsession.SessionsGetRsp
	SessionsDeleteReq    = modeliamsession.SessionsDeleteReq
	SessionsDeleteRsp    = modeliamsession.SessionsDeleteRsp
	SessionsDeleteAllReq = modeliamsession.SessionsDeleteAllReq
	SessionsDeleteAllRsp = modeliamsession.SessionsDeleteAllRsp

	AdminSessionUserView   = modeliamsession.AdminSessionUserView
	AdminSessionsListReq   = modeliamsession.AdminSessionsListReq
	AdminSessionsListRsp   = modeliamsession.AdminSessionsListRsp
	AdminSessionsGetReq    = modeliamsession.AdminSessionsGetReq
	AdminSessionsGetRsp    = modeliamsession.AdminSessionsGetRsp
	AdminSessionsDeleteReq = modeliamsession.AdminSessionsDeleteReq
	AdminSessionsDeleteRsp = modeliamsession.AdminSessionsDeleteRsp

	AdminUserSessionsListReq   = modeliamsession.AdminUserSessionsListReq
	AdminUserSessionsListRsp   = modeliamsession.AdminUserSessionsListRsp
	AdminUserSessionsDeleteReq = modeliamsession.AdminUserSessionsDeleteReq
	AdminUserSessionsDeleteRsp = modeliamsession.AdminUserSessionsDeleteRsp

	EmailChangeConfirmReq = modeliamemail.ChangeConfirmReq
	EmailChangeConfirmRsp = modeliamemail.ChangeConfirmRsp
	EmailChangeCancelReq  = modeliamemail.ChangeCancelReq
	EmailChangeCancelRsp  = modeliamemail.ChangeCancelRsp
	EmailChangeRequestReq = modeliamemail.ChangeRequestReq
	EmailChangeRequestRsp = modeliamemail.ChangeRequestRsp
	EmailChangeResendReq  = modeliamemail.ChangeResendReq
	EmailChangeResendRsp  = modeliamemail.ChangeResendRsp

	EmailPasswordResetConfirmReq = modeliamemail.PasswordResetConfirmReq
	EmailPasswordResetConfirmRsp = modeliamemail.PasswordResetConfirmRsp
	EmailPasswordResetRequestReq = modeliamemail.PasswordResetRequestReq
	EmailPasswordResetRequestRsp = modeliamemail.PasswordResetRequestRsp

	EmailVerificationConfirmReq = modeliamemail.VerificationConfirmReq
	EmailVerificationConfirmRsp = modeliamemail.VerificationConfirmRsp
	EmailVerificationResendReq  = modeliamemail.VerificationResendReq
	EmailVerificationResendRsp  = modeliamemail.VerificationResendRsp
	EmailVerificationRequestReq = modeliamemail.VerificationRequestReq
	EmailVerificationRequestRsp = modeliamemail.VerificationRequestRsp
)
