package modeliam

import (
	. "github.com/forbearing/gst/dsl"
	"github.com/forbearing/gst/model"
)

type Offline struct {
	model.Empty
}

type OfflineReq struct {
	UserID string `json:"user_id"`
}

func (Offline) Design() {
	Route("offline", func() {
		Create(func() {
			Enabled(true)
			Service(true)
			Payload[*OfflineReq]()
		})
	})
}
