package helloworld

import (
	"github.com/forbearing/gst/model"
	"github.com/forbearing/gst/service"
	"github.com/forbearing/gst/types"
)

var _ types.Module[*Helloworld2, *Helloworld2, *Helloworld2] = (*Helloworld2Module)(nil)

type Helloworld2 struct {
	Before string `json:"before" schema:"before"`
	After  string `json:"after" schema:"after"`

	model.Base
}

type Service2 struct {
	service.Base[*Helloworld2, *Helloworld2, *Helloworld2]
}

type Helloworld2Module struct{}

func (*Helloworld2Module) Service() types.Service[*Helloworld2, *Helloworld2, *Helloworld2] {
	return &Service2{}
}

func (*Helloworld2Module) Route() string {
	return "hello-world2"
}

// Param returns the route parameter identifier.
// returns empty string to use default "id".
func (*Helloworld2Module) Param() string {
	return ""
}

func (*Helloworld2Module) Pub() bool {
	return false
}
