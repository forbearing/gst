package serviceaichat

import (
	modelaichat "github.com/forbearing/gst/internal/model/aichat"
	"github.com/forbearing/gst/service"
)

type ModelService struct {
	service.Base[*modelaichat.Model, *modelaichat.Model, *modelaichat.Model]
}
