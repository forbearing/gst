package twofa

import (
	modeltwofa "github.com/forbearing/gst/internal/model/twofa"
	servicetwofa "github.com/forbearing/gst/internal/service/twofa"
	"github.com/forbearing/gst/types"
)

var _ types.Module[*TOTPVerify, *TOTPVerifyReq, *TOTPVerifyRsp] = (*TOTPVerifyModule)(nil)

type (
	TOTPVerify        = modeltwofa.TOTPVerify
	TOTPVerifyReq     = modeltwofa.TOTPVerifyReq
	TOTPVerifyRsp     = modeltwofa.TOTPVerifyRsp
	TOTPVerifyService = servicetwofa.TOTPVerifyService
	TOTPVerifyModule  struct{}
)

func (*TOTPVerifyModule) Service() types.Service[*TOTPVerify, *TOTPVerifyReq, *TOTPVerifyRsp] {
	return &TOTPVerifyService{}
}
func (*TOTPVerifyModule) Route() string { return "2fa/totp/verify" }
func (*TOTPVerifyModule) Pub() bool     { return false }
func (*TOTPVerifyModule) Param() string { return "id" }
