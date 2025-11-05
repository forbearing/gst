package helloworld

import (
	"github.com/forbearing/gst/module"
	"github.com/forbearing/gst/types/consts"
)

// Register registers two modules: Helloworld and Helloworld2.
// helloworld demo just used for demo, that not contains any business logic.
//
// Models:
//   - Helloworld
//   - Helloworld2
//
// Routes:
//   - /api/hello-world
//   - /api/hello-world2
func Register() {
	module.Use[
		*Helloworld,
		*Req,
		*Rsp,
		*Service](
		&Module{},
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

	module.Use[
		*Helloworld2,
		*Helloworld2,
		*Helloworld2,
		*Service2](
		&Module2{},
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
