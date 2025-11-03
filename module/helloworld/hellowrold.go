package helloworld

import (
	"github.com/forbearing/gst/model"
	"github.com/forbearing/gst/service"
	"github.com/forbearing/gst/types"
)

var _ types.Module[*Helloworld, *Req, *Rsp] = (*HelloworldModule)(nil)

// Helloworld is the model definition.
type Helloworld struct {
	Hello string `json:"hello" schema:"hello"`
	World string `json:"world" schema:"world"`

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

// HelloworldModule implements the `types.Module` interface.
type HelloworldModule struct{}

func (HelloworldModule) Service() types.Service[*Helloworld, *Req, *Rsp] {
	return &Service{}
}
func (HelloworldModule) Pub() bool     { return false }
func (HelloworldModule) Route() string { return "hello-world" }
func (HelloworldModule) Param() string { return "id" }
