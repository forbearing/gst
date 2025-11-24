package authz

import (
	modelauthz "github.com/forbearing/gst/internal/model/authz"
	serviceauthz "github.com/forbearing/gst/internal/service/authz"
	"github.com/forbearing/gst/types"
)

var _ types.Module[*Button, *Button, *Button] = (*ButtonModule)(nil)

type (
	Button        = modelauthz.Button
	ButtonService = serviceauthz.ButtonService
	ButtonModule  struct{}
)

func (*ButtonModule) Service() types.Service[*Button, *Button, *Button] {
	return &ButtonService{}
}
func (*ButtonModule) Route() string { return "buttons" }
func (*ButtonModule) Pub() bool     { return false }
func (*ButtonModule) Param() string { return "id" }
