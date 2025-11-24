package serviceiam

import (
	modeliam "github.com/forbearing/gst/internal/model/iam"
	"github.com/forbearing/gst/service"
)

type TenantService struct {
	service.Base[*modeliam.Tenant, *modeliam.Tenant, *modeliam.Tenant]
}
