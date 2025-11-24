package servicelogmgmt

import (
	modellogmgmt "github.com/forbearing/gst/internal/model/logmgmt"
	"github.com/forbearing/gst/service"
)

type LoginLogService struct {
	service.Base[*modellogmgmt.LoginLog, *modellogmgmt.LoginLog, *modellogmgmt.LoginLog]
}
