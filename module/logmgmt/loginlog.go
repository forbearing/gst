package logmgmt

import (
	modellogmgmt "github.com/forbearing/gst/internal/model/logmgmt"
	servicelogmgmt "github.com/forbearing/gst/internal/service/logmgmt"
	"github.com/forbearing/gst/types"
)

var _ types.Module[*LoginLog, *LoginLog, *LoginLog] = (*LoginLogModule)(nil)

type (
	LoginStatus = modellogmgmt.LoginStatus

	LoginLog        = modellogmgmt.LoginLog
	LoginLogService = servicelogmgmt.LoginLogService
	LoginLogModule  struct{}
)

const (
	LoginStatusSuccess = modellogmgmt.LoginStatusSuccess
	LoginStatusFailure = modellogmgmt.LoginStatusFailure
	LoginStatusLogout  = modellogmgmt.LoginStatusLogout
)

func (*LoginLogModule) Service() types.Service[*LoginLog, *LoginLog, *LoginLog] {
	return &LoginLogService{}
}
func (*LoginLogModule) Route() string { return "/log/loginlog" }
func (*LoginLogModule) Pub() bool     { return false }
func (*LoginLogModule) Param() string { return "id" }
