package iam

import (
	modeliam "github.com/forbearing/gst/internal/model/iam"
	modeliamaccount "github.com/forbearing/gst/internal/model/iam/account"
	modeliamemail "github.com/forbearing/gst/internal/model/iam/email"
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

	Heartbeat  = modeliam.Heartbeat
	MeRsp      = modeliam.MeRsp
	OnlineUser = modeliam.OnlineUser
	OfflineReq = modeliam.OfflineReq

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
