package serviceaichat

import (
	modelaichat "github.com/forbearing/gst/internal/model/aichat"
	"github.com/forbearing/gst/service"
)

type ProviderService struct {
	service.Base[*modelaichat.Provider, *modelaichat.Provider, *modelaichat.Provider]
}
