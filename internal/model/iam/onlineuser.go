package modeliam

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"

	. "github.com/forbearing/gst/dsl"
	"github.com/forbearing/gst/model"
	"github.com/forbearing/gst/types"
)

type OnlineUser struct {
	// User Info
	UserID   string `json:"user_id,omitempty" schema:"user_id"`
	Username string `json:"username,omitempty" schema:"username"`
	ClientIP string `json:"client_ip,omitempty" schema:"client_ip"`

	// User Agent info.
	Source   string `json:"source" schema:"source"`
	Platform string `json:"platform" schema:"platform"`
	Engine   string `json:"engine" schema:"engine"`
	Browser  string `json:"browser" schema:"browser"`

	model.Base
}

type (
	OnlineUserReq        struct{}
	OnlineUserOfflineReq struct {
		UserID string `json:"user_id"`
	}
)

func (OnlineUser) Design() {
	Migrate(true)
	// api to receive heartbeat from frontend
	Route("heartbeat", func() {
		Create(func() {
			Enabled(true)
			Service(true)
			Payload[OnlineUserReq]()
		})
	})
	Route("online-users", func() {
		List(func() {
			Enabled(true)
		})
	})
}

func (ou *OnlineUser) CreateBefore(ctx *types.ModelContext) error { return ou.validate(ctx) }
func (ou *OnlineUser) UpdateBefore(ctx *types.ModelContext) error { return ou.validate(ctx) }

func (ou *OnlineUser) validate(_ *types.ModelContext) error {
	// Uniquely identifies an active online user by combining userID, clientIP and source(UserAgent).
	sum := sha256.Sum256(fmt.Appendf(nil, "%s:%s:%s", ou.UserID, ou.ClientIP, ou.Source))
	id := hex.EncodeToString(sum[:])
	ou.SetID(id)

	return nil
}
