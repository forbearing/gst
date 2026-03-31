package iam

import (
	modeliam "github.com/forbearing/gst/internal/model/iam"
	serviceiam "github.com/forbearing/gst/internal/service/iam"
	"github.com/forbearing/gst/types"
)

var _ types.Module[*Offline, *OfflineReq, *Offline] = (*OfflineModule)(nil)

type (
	Offline       = modeliam.Offline
	OfflineReq    = modeliam.OfflineReq
	OfflineModule struct{}
)

func (*OfflineModule) Service() types.Service[*Offline, *OfflineReq, *Offline] {
	return &serviceiam.OfflineService{}
}
func (*OfflineModule) Route() string { return "/offline" }
func (*OfflineModule) Pub() bool     { return false }
func (*OfflineModule) Param() string { return "id" }
