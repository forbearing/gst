package helloworld

import (
	"github.com/forbearing/gst/model"
	"github.com/forbearing/gst/service"
	"github.com/forbearing/gst/types"
)

var _ types.Plugin[*Helloworld, *Req, *Rsp] = (*HelloworldPlugin)(nil)

// Helloworld is the model definition.
type Helloworld struct {
	model.Base
}

// Req is the custom request type.
type Req struct {
	Field1 string
	Field2 int
}

// Rsp is the custom response type.
type Rsp struct {
	Field3 string
	Field4 int
}

// Service implements the `types.Service` interface.
type Service struct {
	service.Base[*Helloworld, *Req, *Rsp]
}

// HelloworldPlugin implements the `types.Plugin` interface.
type HelloworldPlugin struct{}

func (HelloworldPlugin) Service() types.Service[*Helloworld, *Req, *Rsp] {
	return &Service{}
}
