package aichat

import (
	modelaichat "github.com/forbearing/gst/internal/model/aichat"
	serviceaichat "github.com/forbearing/gst/internal/service/aichat"
	"github.com/forbearing/gst/types"
)

var _ types.Module[*Provider, *Provider, *Provider] = (*ProviderModule)(nil)

type (
	Provider        = modelaichat.Provider
	ProviderService = serviceaichat.ProviderService
	ProviderModule  struct{}
)

func (*ProviderModule) Service() types.Service[*Provider, *Provider, *Provider] {
	return &ProviderService{}
}
func (*ProviderModule) Route() string { return "/ai/providers" }
func (*ProviderModule) Pub() bool     { return false }
func (*ProviderModule) Param() string { return "id" }
