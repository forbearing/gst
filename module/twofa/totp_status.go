package twofa

import (
	modeltwofa "github.com/forbearing/gst/internal/model/twofa"
	servicetwofa "github.com/forbearing/gst/internal/service/twofa"
	"github.com/forbearing/gst/types"
)

var _ types.Module[*TOTPStatus, *TOTPStatus, *TOTPStatusRsp] = (*TOTPStatusModule)(nil)

type (
	TOTPStatus        = modeltwofa.TOTPStatus
	TOTPStatusRsp     = modeltwofa.TOTPStatusRsp
	TOTPStatusService = servicetwofa.TOTPStatusService
	TOTPStatusModule  struct{}
)

func (*TOTPStatusModule) Service() types.Service[*TOTPStatus, *TOTPStatus, *TOTPStatusRsp] {
	return &TOTPStatusService{}
}
func (*TOTPStatusModule) Route() string { return "2fa/totp/status" }
func (*TOTPStatusModule) Pub() bool     { return false }
func (*TOTPStatusModule) Param() string { return "id" }
