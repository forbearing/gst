package iam

import (
	modeliam "github.com/forbearing/gst/internal/model/iam"
	"github.com/forbearing/gst/types"
)

var _ types.Module[*Group, *Group, *Group] = (*GroupModule)(nil)

type (
	Group        = modeliam.Group
	GroupService = modeliam.GroupService
	GroupModule  struct{}
)

func (*GroupModule) Service() types.Service[*Group, *Group, *Group] {
	return &GroupService{}
}
func (*GroupModule) Route() string { return "/iam/groups" }
func (*GroupModule) Pub() bool     { return false }
func (*GroupModule) Param() string { return "id" }
