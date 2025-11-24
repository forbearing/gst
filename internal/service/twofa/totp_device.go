package servicetwofa

import (
	modeltwofa "github.com/forbearing/gst/internal/model/twofa"
	"github.com/forbearing/gst/service"
)

type TOTPDeviceService struct {
	service.Base[*modeltwofa.TOTPDevice, *modeltwofa.TOTPDevice, *modeltwofa.TOTPDevice]
}
