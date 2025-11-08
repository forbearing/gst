package authz

import (
	"github.com/forbearing/gst/module"
	"github.com/forbearing/gst/service"
	"github.com/forbearing/gst/types/consts"
)

func Register() {
	module.Use[
		*Permission,
		*Permission,
		*Permission,
		service.Base[*Permission, *Permission, *Permission]](
		&PermissionModule{},
		consts.PHASE_LIST,
		consts.PHASE_GET)
}
