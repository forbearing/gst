package twofa

import (
	modeltwofa "github.com/forbearing/gst/internal/model/twofa"
	servicetwofa "github.com/forbearing/gst/internal/service/twofa"
	"github.com/forbearing/gst/types"
)

var _ types.Module[*TOTPCheck, *TOTPCheckReq, *TOTPCheckRsp] = (*TOTPCheckModule)(nil)

type (
	TOTPCheck        = modeltwofa.TOTPCheck
	TOTPCheckReq     = modeltwofa.TOTPCheckReq
	TOTPCheckRsp     = modeltwofa.TOTPCheckRsp
	TOTPCheckService = servicetwofa.TOTPCheckService
	TOTPCheckModule  struct{}
)

func (*TOTPCheckModule) Service() types.Service[*TOTPCheck, *TOTPCheckReq, *TOTPCheckRsp] {
	return &TOTPCheckService{}
}
func (*TOTPCheckModule) Route() string { return "2fa/totp/check" }
func (*TOTPCheckModule) Pub() bool     { return true }
func (*TOTPCheckModule) Param() string { return "id" }
