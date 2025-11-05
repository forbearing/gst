package twofa

import (
	"github.com/forbearing/gst/module"
	"github.com/forbearing/gst/types/consts"
)

// Register registers the models: TOTPBind
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
}
