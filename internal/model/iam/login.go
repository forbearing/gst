package modeliam

import (
	"github.com/forbearing/gst/dsl"
	"github.com/forbearing/gst/model"
)

const SessionNamespace = "identity:session"

type Login struct {
	model.Empty
}

type LoginReq struct {
	Username   string `json:"username"`
	Password   string `json:"password"`
	TOTPCode   string `json:"totp_code,omitempty"`   // Optional TOTP code
	BackupCode string `json:"backup_code,omitempty"` // Optional backup code
}

type LoginRsp struct {
	SessionID string `json:"session_id"`
}

func (Login) Design() {
	dsl.Create(func() {
		dsl.Enabled(true)
		dsl.Service(true)
		dsl.Public(true)
		dsl.Payload[*LoginReq]()
		dsl.Result[*LoginRsp]()
	})
}
