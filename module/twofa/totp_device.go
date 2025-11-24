package twofa

import (
	modeltwofa "github.com/forbearing/gst/internal/model/twofa"
	servicetwofa "github.com/forbearing/gst/internal/service/twofa"
	"github.com/forbearing/gst/types"
)

var _ types.Module[*TOTPDevice, *TOTPDevice, *TOTPDevice] = (*TOTPDeviceModule)(nil)

type (
	TOTPDevice        = modeltwofa.TOTPDevice
	TOTPDeviceService = servicetwofa.TOTPDeviceService
	TOTPDeviceModule  struct{}
)

func (*TOTPDeviceModule) Service() types.Service[*TOTPDevice, *TOTPDevice, *TOTPDevice] {
	return &TOTPDeviceService{}
}
func (*TOTPDeviceModule) Route() string { return "2fa/totp/devices" }
func (*TOTPDeviceModule) Pub() bool     { return false }
func (*TOTPDeviceModule) Param() string { return "id" }
