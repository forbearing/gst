package servicelogmgmt

import (
	modellogmgmt "github.com/forbearing/gst/internal/model/logmgmt"
	"github.com/forbearing/gst/service"
)

type OperationLogService struct {
	service.Base[*modellogmgmt.OperationLog, *modellogmgmt.OperationLog, *modellogmgmt.OperationLog]
}
