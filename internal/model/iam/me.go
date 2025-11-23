package modeliam

import (
	"github.com/forbearing/gst/dsl"
	"github.com/forbearing/gst/model"
	"github.com/forbearing/gst/service"
)

type Me struct {
	model.Empty
}

type MeRsp = map[string]any

func (Me) Design() {
	dsl.List(func() {
		dsl.Enabled(true)
		dsl.Service(true)
		dsl.Public(true)
		dsl.Result[MeRsp]()
	})
}

type MeService struct {
	service.Base[*Me, *Me, *MeRsp]
}
