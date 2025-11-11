package authz

import (
	modelauthz "github.com/forbearing/gst/internal/model/authz"
	"github.com/forbearing/gst/service"
	"github.com/forbearing/gst/types"
)

var _ types.Module[*Button, *Button, *Button] = (*ButtonModule)(nil)

type ButtonModule struct{}

func (*ButtonModule) Service() types.Service[*Button, *Button, *Button] {
	return &buttonservice{}
}
func (*ButtonModule) Route() string { return "buttons" }
func (*ButtonModule) Pub() bool     { return false }
func (*ButtonModule) Param() string { return "id" }

type Button = modelauthz.Button

type buttonservice struct {
	service.Base[*Button, *Button, *Button]
}
