// Package model provides ...
package model

import "go.uber.org/zap/zapcore"

type UserInfo struct {
	AccessToken      string `json:"access_token,omitempty"`
	TokenType        string `json:"token_type,omitempty"`         // token 类型
	ExpiresIn        int    `json:"expires_in,omitempty"`         // `access_token`的有效期，单位: 秒
	Name             string `json:"name,omitempty"`               // 用户姓名
	EnName           string `json:"en_name,omitempty"`            // 用户英文名称
	AvatarURL        string `json:"avatar_url,omitempty"`         // 用户头像
	AvatarThumb      string `json:"avatar_thumb,omitempty"`       // 用户头像 72x72
	AvatarMiddle     string `json:"avatar_middle,omitempty"`      // 用户头像 240x240
	AvatarBig        string `json:"avatar_big,omitempty"`         // 用户头像 640x640
	OpenID           string `json:"open_id,omitempty"`            // 用户在应用内的唯一标识
	UnionID          string `json:"union_id,omitempty"`           // 用户统一ID
	Email            string `json:"email,omitempty"`              // 用户邮箱
	EnterpriseEmail  string `json:"enterprise_email,omitempty"`   // 企业邮箱，请先确保已在管理后台启用飞书邮箱服务
	UserID           string `json:"user_id,omitempty"`            // 用户 user_id
	Mobile           string `json:"mobile,omitempty"`             // 用户手机号
	TenantKey        string `json:"tenant_key,omitempty"`         // 当前企业标识
	RefreshExpiresIn int    `json:"refresh_expires_in,omitempty"` // `refresh_token` 的有效期，单位: 秒
	RefreshToken     string `json:"refresh_token,omitempty"`
	Sid              string `json:"sid,omitempty"` // 用户当前登录态session的唯一标识，为空则不返回

	Base
}

func (u *UserInfo) MarshalLogObject(enc zapcore.ObjectEncoder) error {
	if u == nil {
		return nil
	}
	enc.AddString("access_token", u.AccessToken)
	enc.AddString("token_type", u.TokenType)
	enc.AddInt("expires_in", u.ExpiresIn)
	enc.AddString("name", u.Name)
	enc.AddString("en_name", u.EnName)
	enc.AddString("avatar_url", u.AvatarURL)
	enc.AddString("avatar_thumb", u.AvatarThumb)
	enc.AddString("avatar_middle", u.AvatarMiddle)
	enc.AddString("avatar_big", u.AvatarBig)
	enc.AddString("open_id", u.OpenID)
	enc.AddString("union_id", u.UnionID)
	enc.AddString("email", u.Email)
	enc.AddString("enterprise_email", u.EnterpriseEmail)
	enc.AddString("user_id", u.UserID)
	enc.AddString("mobile", u.Mobile)
	enc.AddString("tenant_key", u.TenantKey)
	enc.AddInt("refresh_expires_in", u.RefreshExpiresIn)
	enc.AddString("refresh_token", u.RefreshToken)
	enc.AddString("sid", u.Sid)
	enc.AddObject("base", &u.Base) //nolint:errcheck
	return nil
}
