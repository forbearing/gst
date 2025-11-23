package iam

import (
	modeliam "github.com/forbearing/gst/internal/model/iam"
	"github.com/forbearing/gst/types"
)

var _ types.Module[*Me, *Me, *MeRsp] = (*MeModule)(nil)

type (
	Me        = modeliam.Me
	MeRsp     = modeliam.MeRsp
	MeService = modeliam.MeService
	MeModule  struct{}
)

func (*MeModule) Service() types.Service[*Me, *Me, *MeRsp] {
	return &MeService{}
}
func (*MeModule) Route() string { return "/iam/me" }
func (*MeModule) Pub() bool     { return false }
func (*MeModule) Param() string { return "id" }
