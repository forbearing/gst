package modeliam

import (
	. "github.com/forbearing/gst/dsl"
	"github.com/forbearing/gst/model"
)

type OnlineUserOffline struct {
	model.Empty
}

func (OnlineUserOffline) Design() {
	Route("online-users/offline", func() {
		Create(func() {
			Enabled(true)
			Service(true)
			Payload[*OnlineUserOfflineReq]()
		})
	})
}
