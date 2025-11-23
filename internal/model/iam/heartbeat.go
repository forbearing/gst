package modeliam

import (
	. "github.com/forbearing/gst/dsl"
	"github.com/forbearing/gst/model"
	"github.com/forbearing/gst/service"
)

type Heartbeat struct {
	model.Empty
}

func (Heartbeat) Design() {
	// Route specific the api to receive heartbeat from frontend or client.
	Route("heartbeat", func() {
		Create(func() {
			Enabled(true)
			Service(true)
		})
	})
}

type HeartbeatService struct {
	service.Base[*Heartbeat, *Heartbeat, *Heartbeat]
}
