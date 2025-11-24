package authz

import (
	modelauthz "github.com/forbearing/gst/internal/model/authz"
	serviceauthz "github.com/forbearing/gst/internal/service/authz"
	"github.com/forbearing/gst/types"
)

var _ types.Module[*Menu, *Menu, *Menu] = (*MenuModule)(nil)

type (
	Menu        = modelauthz.Menu
	MenuService = serviceauthz.MenuService
	MenuModule  struct{}
)

func (*MenuModule) Service() types.Service[*Menu, *Menu, *Menu] {
	return &MenuService{}
}
func (*MenuModule) Route() string { return "menus" }
func (*MenuModule) Pub() bool     { return false }
func (*MenuModule) Param() string { return "id" }
