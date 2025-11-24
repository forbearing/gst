package modeltwofa

import (
	. "github.com/forbearing/gst/dsl"
	"github.com/forbearing/gst/model"
)

// TOTPCheck 检查用户是否需要 2FA 验证
type TOTPCheck struct {
	model.Empty
}

type TOTPCheckReq struct {
	Username string `json:"username" validate:"required"` // 用户名
	Password string `json:"password" validate:"required"` // 密码（用于验证权限）
}

type TOTPCheckRsp struct {
	Requires2FA bool   `json:"requires_2fa"` // 是否需要 2FA 验证
	Message     string `json:"message"`      // 响应消息
}

func (TOTPCheck) Design() {
	Route("2fa/totp/check", func() {
		Create(func() {
			Enabled(true)
			Service(true)
			Public(true) // 公开接口，用于登录前检查
			Payload[*TOTPCheckReq]()
			Result[*TOTPCheckRsp]()
		})
	})
}
