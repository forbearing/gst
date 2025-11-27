package logmgmt

import (
	"os"

	"github.com/forbearing/gst/config"
	"github.com/forbearing/gst/cronjob"
	cronjoblogmgmt "github.com/forbearing/gst/internal/cronjob/logmgmt"
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
//   - GET /api/log/loginlog
//   - GET /api/log/loginlog/:id
//   - GET /api/log/operationlog
//   - GET /api/log/operationlog/:id
//
// Cronjob:
//   - cleanup operationlog and loginlog hourly.
//
// Enable Audit to records all operation logs.
func Register() {
	// enable audit function to records the operation logs.
	os.Setenv(config.AUDIT_ENABLE, "true")

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

	cronjob.Register(cronjoblogmgmt.Cleanup, "0 0 * * * *", "cleanup operationlog and loginlog hourly")
}
