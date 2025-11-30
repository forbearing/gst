package aichat

import (
	"github.com/forbearing/gst/module"
	"github.com/forbearing/gst/types/consts"
)

func Register() {
	// Register "Model" module.
	module.Use[
		*Model,
		*Model,
		*Model,
		*ModelService](
		&ModelModule{},
		consts.PHASE_CREATE,
		consts.PHASE_DELETE,
		consts.PHASE_UPDATE,
		consts.PHASE_PATCH,
		consts.PHASE_LIST,
		consts.PHASE_GET,
		consts.PHASE_CREATE_MANY,
		consts.PHASE_DELETE_MANY,
		consts.PHASE_UPDATE_MANY,
		consts.PHASE_PATCH_MANY,
	)

	// Register "Provider" module.
	module.Use[
		*Provider,
		*Provider,
		*Provider,
		*ProviderService](
		&ProviderModule{},
		consts.PHASE_CREATE,
		consts.PHASE_DELETE,
		consts.PHASE_UPDATE,
		consts.PHASE_PATCH,
		consts.PHASE_LIST,
		consts.PHASE_GET,
		consts.PHASE_CREATE_MANY,
		consts.PHASE_DELETE_MANY,
		consts.PHASE_UPDATE_MANY,
		consts.PHASE_PATCH_MANY,
	)
}
