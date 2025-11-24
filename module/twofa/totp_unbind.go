package twofa

import (
	modeltwofa "github.com/forbearing/gst/internal/model/twofa"
	servicetwofa "github.com/forbearing/gst/internal/service/twofa"
	"github.com/forbearing/gst/types"
)

var _ types.Module[*TOTPUnbind, *TOTPUnbindReq, *TOTPUnbindRsp] = (*TOTPUnbindModule)(nil)

type (
	TOTPUnbind        = modeltwofa.TOTPUnbind
	TOTPUnbindReq     = modeltwofa.TOTPUnbindReq
	TOTPUnbindRsp     = modeltwofa.TOTPUnbindRsp
	TOTPUnbindService = servicetwofa.TOTPUnbindService
	TOTPUnbindModule  struct{}
)

func (*TOTPUnbindModule) Service() types.Service[*TOTPUnbind, *TOTPUnbindReq, *TOTPUnbindRsp] {
	return &TOTPUnbindService{}
}
func (*TOTPUnbindModule) Route() string { return "2fa/totp/unbind" }
func (*TOTPUnbindModule) Pub() bool     { return false }
func (*TOTPUnbindModule) Param() string { return "id" }
