package version

import (
	"github.com/forbearing/gst/module"
	"github.com/forbearing/gst/types/consts"
)

// Register registers the version module.
//
// Modals and Result:
//   - Version, VersionRsp
//
// Routes:
//   - GET /api/version
func Register() {
	module.Use[
		*Version,
		*Version,
		*VersionRsp,
		*VersionService](
		&VersionModule{},
		consts.PHASE_LIST,
	)
}
