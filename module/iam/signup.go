package iam

import (
	modeliam "github.com/forbearing/gst/internal/model/iam"
	serviceiam "github.com/forbearing/gst/internal/service/iam"
	"github.com/forbearing/gst/types"
)

var _ types.Module[*Signup, *SignupReq, *SignupRsp] = (*SignupModule)(nil)

type (
	Signup        = modeliam.Signup
	SignupReq     = modeliam.SignupReq
	SignupRsp     = modeliam.SignupRsp
	SignupService = serviceiam.SignupService
	SignupModule  struct{}
)

func (*SignupModule) Service() types.Service[*Signup, *SignupReq, *SignupRsp] {
	return &SignupService{}
}

func (*SignupModule) Route() string { return "/signup" }
func (*SignupModule) Pub() bool     { return true }
func (*SignupModule) Param() string { return "id" }
