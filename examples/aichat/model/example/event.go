package example

import (
	. "github.com/forbearing/gst/dsl"
	"github.com/forbearing/gst/model"
)

type Event struct {
	model.Empty
}

func (Event) Design() {
	Route("/event", func() {
		List(func() {
			Enabled(true)
			Service(true)
		})
	})
}
