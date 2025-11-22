package modeliam

import "github.com/forbearing/gst/model"

// Session 删除缓存中的 session 切调用 Keycloak logout，退出登录
type Session struct {
	UserID   string `json:"user_id"`
	Username string `json:"username"`
	Email    string `json:"email"`

	Platform    string `json:"platform"`
	OS          string `json:"os"`
	EngineName  string `json:"engine_name"`
	BrowserName string `json:"browser_name"`

	Token    Token
	UserInfo map[string]any `json:"user_info"`

	model.Base
}
type Token struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`

	ExpiresIn        int `json:"expires_in"`
	RefreshExpiresIn int `json:"refresh_expires_in"`

	TokenType string `json:"token_type"`
	Scope     string `json:"scope"`

	NotBeforePolicy int    `json:"not-before-policy"`
	SessionState    string `json:"session_state"`
}
