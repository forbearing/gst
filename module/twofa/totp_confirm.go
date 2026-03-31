package twofa

import (
	modeltwofa "github.com/forbearing/gst/internal/model/twofa"
	servicetwofa "github.com/forbearing/gst/internal/service/twofa"
	"github.com/forbearing/gst/types"
)

var _ types.Module[*TOTPConfirm, *TOTPConfirmReq, *TOTPConfirmRsp] = (*TOTPConfirmModule)(nil)

type (
	TOTPConfirm       = modeltwofa.TOTPConfirm
	TOTPConfirmReq    = modeltwofa.TOTPConfirmReq
	TOTPConfirmRsp    = modeltwofa.TOTPConfirmRsp
	TOTPConfirmModule struct{}
)

func (*TOTPConfirmModule) Service() types.Service[*TOTPConfirm, *TOTPConfirmReq, *TOTPConfirmRsp] {
	return &servicetwofa.TOTPConfirmService{}
}
func (*TOTPConfirmModule) Route() string { return "2fa/totp/confirm" }
func (*TOTPConfirmModule) Pub() bool     { return false }
func (*TOTPConfirmModule) Param() string { return "id" }
