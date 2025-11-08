package logmgmt

import (
	modellog "github.com/forbearing/gst/internal/model/log"
	"github.com/forbearing/gst/service"
	"github.com/forbearing/gst/types"
)

var _ types.Module[*OperationLog, *OperationLog, *OperationLog] = (*operationLogModule)(nil)

type OperationLog = modellog.OperationLog

type operationLogService struct {
	service.Base[*OperationLog, *OperationLog, *OperationLog]
}

type operationLogModule struct{}

func (*operationLogModule) Service() types.Service[*OperationLog, *OperationLog, *OperationLog] {
	return &operationLogService{}
}

func (*operationLogModule) Pub() bool     { return false }
func (*operationLogModule) Route() string { return "/log/operationlog" }
func (*operationLogModule) Param() string { return "id" }
