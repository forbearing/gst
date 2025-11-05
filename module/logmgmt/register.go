package logmgmt

import (
	"github.com/forbearing/gst/cronjob"
	"github.com/forbearing/gst/module"
	"github.com/forbearing/gst/types/consts"
)

// Register registers two modules: LoginLog and OperationLog.
//
// Models:
//   - LoginLog
//   - OperationLog
//
// Routes:
//   - /api/log/loginlog
//   - /api/log/operationlog
//
// Cronjob:
//   - cleanup operationlog and loginlog hourly.
func Register() {
	module.Use[*LoginLog,
		*LoginLog,
		*LoginLog,
		*loginLogService](
		&loginLogModule{},
		consts.PHASE_LIST,
		consts.PHASE_GET,
	)

	module.Use[
		*OperationLog,
		*OperationLog,
		*OperationLog,
		*operationLogService](
		&operationLogModule{},
		consts.PHASE_LIST,
		consts.PHASE_GET,
	)

	cronjob.Register(cleanupLogs, "0 0 * * * *", "cleanup operationlog and loginlog hourly")
}
