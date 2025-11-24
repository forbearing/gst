package logmgmt

import (
	modellogmgmt "github.com/forbearing/gst/internal/model/logmgmt"
	servicelogmgmt "github.com/forbearing/gst/internal/service/logmgmt"
	"github.com/forbearing/gst/types"
)

var _ types.Module[*OperationLog, *OperationLog, *OperationLog] = (*OperationLogModule)(nil)

type (
	OperationLog        = modellogmgmt.OperationLog
	OperationLogService = servicelogmgmt.OperationLogService
	OperationLogModule  struct{}
)

func (*OperationLogModule) Service() types.Service[*OperationLog, *OperationLog, *OperationLog] {
	return &OperationLogService{}
}

func (*OperationLogModule) Route() string { return "/log/operationlog" }
func (*OperationLogModule) Pub() bool     { return false }
func (*OperationLogModule) Param() string { return "id" }
