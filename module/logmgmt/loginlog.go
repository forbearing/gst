package logmgmt

import (
	. "github.com/forbearing/gst/dsl"
	"github.com/forbearing/gst/model"
	"github.com/forbearing/gst/service"
	"github.com/forbearing/gst/types"
)

var _ types.Module[*LoginLog, *LoginLog, *LoginLog] = (*loginLogModule)(nil)

type LoginStatus string

const (
	LoginStatusSuccess = "success"
	LoginStatusFailure = "failure"
	LoginStatusLogout  = "logout"
)

type LoginLog struct {
	// User Info
	UserID   string      `json:"user_id,omitempty" schema:"user_id"`
	Username string      `json:"username,omitempty" schema:"username"`
	ClientIP string      `json:"client_ip,omitempty" schema:"client_ip"`
	Status   LoginStatus `json:"status,omitempty" schema:"status"`

	// User Agent info
	Source   string `json:"source" schema:"source"`
	Platform string `json:"platform" schema:"platform"`
	Engine   string `json:"engine" schema:"engine"`
	Browser  string `json:"browser" schema:"browser"`

	model.Base
}

func (LoginLog) Design() {
	Migrate(true)
	List(func() {
		Enabled(true)
	})
	Get(func() {
		Enabled(true)
	})
}

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
