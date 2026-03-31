package iam

import (
	modeliam "github.com/forbearing/gst/internal/model/iam"
	serviceiam "github.com/forbearing/gst/internal/service/iam"
	"github.com/forbearing/gst/types"
)

var _ types.Module[*Tenant, *Tenant, *Tenant] = (*TenantModule)(nil)

type (
	Tenant       = modeliam.Tenant
	TenantModule struct{}
)

func (*TenantModule) Service() types.Service[*Tenant, *Tenant, *Tenant] {
	return &serviceiam.TenantService{}
}
func (*TenantModule) Route() string { return "/iam/tenants" }
func (*TenantModule) Pub() bool     { return false }
func (*TenantModule) Param() string { return "id" }
