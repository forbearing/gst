package twofa

import (
	"time"

	. "github.com/forbearing/gst/dsl"
	"github.com/forbearing/gst/model"
	"github.com/forbearing/gst/service"
	"github.com/forbearing/gst/types"
	"gorm.io/datatypes"
)

var _ types.Module[*TOTPDevice, *TOTPDevice, *TOTPDevice] = (*TOTPDeviceModule)(nil)

type TOTPDeviceModule struct{}

func (*TOTPDeviceModule) Service() types.Service[*TOTPDevice, *TOTPDevice, *TOTPDevice] {
	return &TOTPDeviceService{}
}
func (*TOTPDeviceModule) Route() string { return "2fa/totp/devices" }
func (*TOTPDeviceModule) Pub() bool     { return false }
func (*TOTPDeviceModule) Param() string { return "id" }

// TOTPDevice represents a TOTP device for 2FA
type TOTPDevice struct {
	UserID      string                      `json:"user_id" gorm:"type:varchar(191);not null;index" schema:"user_id"`
	DeviceName  string                      `json:"device_name" gorm:"type:varchar(100);not null" schema:"device_name"`
	Secret      string                      `json:"-" schema:"secret"`       // Base32 encoded secret, not exposed in JSON
	BackupCodes datatypes.JSONSlice[string] `json:"-" schema:"backup_codes"` // JSON array of backup codes
	IsActive    bool                        `json:"is_active" gorm:"default:true" schema:"is_active"`
	LastUsedAt  *time.Time                  `json:"last_used_at" schema:"last_used_at"`

	model.Base
}

func (TOTPDevice) Design() {
	Migrate(true)

	Route("2fa/totp/devices", func() {
		Create(func() {
			Enabled(true)
		})
		Delete(func() {
			Enabled(true)
		})
		Update(func() {
			Enabled(true)
		})
		Patch(func() {
			Enabled(true)
		})
		List(func() {
			Enabled(true)
		})
		Get(func() {
			Enabled(true)
		})
	})
}

type TOTPDeviceService struct {
	service.Base[*TOTPDevice, *TOTPDevice, *TOTPDevice]
}
