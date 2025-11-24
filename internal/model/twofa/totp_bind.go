package modeltwofa

import (
	. "github.com/forbearing/gst/dsl"
	"github.com/forbearing/gst/model"
)

// TOTPBind 绑定 TOTP 设备
type TOTPBind struct {
	model.Empty
}
type TOTPBindRsp struct {
	Secret      string `json:"secret"`        // Base32 编码的 TOTP 密钥
	OtpauthURL  string `json:"otpauth_url"`   // TOTP 认证 URL
	QRCodeImage string `json:"qr_code_image"` // Base64 编码的二维码图片数据
	Issuer      string `json:"issuer"`        // 应用发行者名称（如 "Nebula"）
	AccountName string `json:"account_name"`  // 用户账户名称
}

func (TOTPBind) Design() {
	Route("2fa/totp/bind", func() {
		Create(func() {
			Enabled(true)
			Service(true)
			Result[*TOTPBindRsp]()
		})
	})
}
