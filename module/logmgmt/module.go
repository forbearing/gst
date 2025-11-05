package logmgmt

import (
	"github.com/forbearing/gst/cronjob"
	"github.com/forbearing/gst/module"
	"github.com/forbearing/gst/types/consts"
)

// Register registers two models: LoginLog OperationLog, and their services, routes and cronjob.
func Register() {
	module.Use[*LoginLog,
		*LoginLog,
		*LoginLog,
		*LoginLogService](
		&LoginLogModule{},
		consts.PHASE_LIST,
		consts.PHASE_GET,
	)

	module.Use[
		*OperationLog,
		*OperationLog,
		*OperationLog,
		*OperationLogService](
		&OperationLogModule{},
		consts.PHASE_LIST,
		consts.PHASE_GET,
	)

	cronjob.Register(cleanupLogs, "0 0 * * * *", "cleanup operationlog and loginlog hourly")
}
