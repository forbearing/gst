package iam

import (
	modeliam "github.com/forbearing/gst/internal/model/iam"
	serviceiam "github.com/forbearing/gst/internal/service/iam"
	"github.com/forbearing/gst/types"
)

var _ types.Module[*Heartbeat, *Heartbeat, *Heartbeat] = (*HeartbeatModule)(nil)

type (
	Heartbeat        = modeliam.Heartbeat
	HeartbeatService = serviceiam.HeartbeatService
	HeartbeatModule  struct{}
)

func (*HeartbeatModule) Service() types.Service[*Heartbeat, *Heartbeat, *Heartbeat] {
	return &HeartbeatService{}
}
func (*HeartbeatModule) Route() string { return "/heartbeat" }
func (*HeartbeatModule) Pub() bool     { return false }
func (*HeartbeatModule) Param() string { return "id" }
