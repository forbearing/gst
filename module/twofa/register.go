package twofa

import (
	"github.com/forbearing/gst/module"
	"github.com/forbearing/gst/types/consts"
)

// Register registers the models: TOTPBind, TOTPCheck, TOTPConfirm, TOTPDevice, TOTPStatus, TOTPUnbind and TOTPVerify.
//
// Modules, Payload and Result:
//   - TOTPBind, TOTPBindRsp
//   - TOTPCheck, TOTPCheckReq, TOTPCheckRsp
//   - TOTPConfirm, TOTPConfirmReq, TOTPConfirmRsp
//   - TOTPDevice
//   - TOTPStatus, TOTPStatusRsp
//   - TOTPUnbind, TOTPUnbindReq, TOTPUnbindRsp
//   - TOTPVerify, TOTPVerifyReq, TOTPVerifyRsp
//
// Routes
//   - POST     /api/2fa/totp/bind
//   - POST     /api/2fa/totp/check
//   - POST     /api/2fa/totp/confirm
//   - POST     /api/2fa/totp/status
//   - POST     /api/2fa/totp/unbind
//   - POST     /api/2fa/totp/verify
//   - POST     /api/2fa/totp/devices
//   - DELETE   /api/2fa/totp/devices/:id
//   - PUT      /api/2fa/totp/devices/:id
//   - PATCH    /api/2fa/totp/devices/:id
//   - GET      /api/2fa/totp/devices
//   - GET      /api/2fa/totp/devices/:id
func Register() {
	module.Use[
		*TOTPBind,
		*TOTPBind,
		*TOTPBindRsp,
		*TOTPBindService](
		&TOTPBindModule{},
		consts.PHASE_CREATE,
	)

	module.Use[
		*TOTPCheck,
		*TOTPCheckReq,
		*TOTPCheckRsp,
		*TOTPCheckService](
		&TOTPCheckModule{},
		consts.PHASE_CREATE,
	)

	module.Use[
		*TOTPConfirm,
		*TOTPConfirmReq,
		*TOTPConfirmRsp,
		*TOTPConfirmService](
		&TOTPConfirmModule{},
		consts.PHASE_CREATE,
	)

	module.Use[
		*TOTPDevice,
		*TOTPDevice,
		*TOTPDevice,
		*TOTPDeviceService](
		&TOTPDeviceModule{},
		consts.PHASE_CREATE,
		consts.PHASE_DELETE,
		consts.PHASE_UPDATE,
		consts.PHASE_PATCH,
		consts.PHASE_LIST,
		consts.PHASE_GET,
	)

	module.Use[
		*TOTPStatus,
		*TOTPStatus,
		*TOTPStatusRsp,
		*TOTPStatusService](
		&TOTPStatusModule{},
		consts.PHASE_LIST,
	)

	module.Use[
		*TOTPUnbind,
		*TOTPUnbindReq,
		*TOTPUnbindRsp,
		*TOTPUnbindService](
		&TOTPUnbindModule{},
		consts.PHASE_CREATE,
	)

	module.Use[
		*TOTPVerify,
		*TOTPVerifyReq,
		*TOTPVerifyRsp,
		*TOTPVerifyService](
		&TOTPVerifyModule{},
		consts.PHASE_CREATE)
}
