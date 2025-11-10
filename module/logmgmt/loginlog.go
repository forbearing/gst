package logmgmt

import (
	modellog "github.com/forbearing/gst/internal/model/log"
	"github.com/forbearing/gst/service"
	"github.com/forbearing/gst/types"
)

var _ types.Module[*LoginLog, *LoginLog, *LoginLog] = (*loginLogModule)(nil)

type (
	LoginLog    = modellog.LoginLog
	LoginStatus = modellog.LoginStatus
)

const (
	LoginStatusSuccess = modellog.LoginStatusSuccess
	LoginStatusFailure = modellog.LoginStatusFailure
	LoginStatusLogout  = modellog.LoginStatusLogout
)

type loginLogService struct {
	service.Base[*LoginLog, *LoginLog, *LoginLog]
}

type loginLogModule struct{}

func (*loginLogModule) Service() types.Service[*LoginLog, *LoginLog, *LoginLog] {
	return &loginLogService{}
}
func (*loginLogModule) Pub() bool     { return false }
func (*loginLogModule) Route() string { return "/log/loginlog" }
func (*loginLogModule) Param() string { return "id" }
