package serviceiamtenant

import (
	"net/http"

	"github.com/forbearing/gst/database"
	modeliamtenant "github.com/forbearing/gst/internal/model/iam/tenant"
	modeliamuser "github.com/forbearing/gst/internal/model/iam/user"
	serviceiamsession "github.com/forbearing/gst/internal/service/iam/session"
	"github.com/forbearing/gst/response"
	"github.com/forbearing/gst/service"
	"github.com/forbearing/gst/types"
	"github.com/forbearing/gst/types/consts"
)

type TenantService struct {
	service.Base[*modeliamtenant.Tenant, *modeliamtenant.Tenant, *modeliamtenant.Tenant]
}

func (TenantService) CreateBefore(ctx *types.ServiceContext, _ *modeliamtenant.Tenant) error {
	return ensureTenantModuleSuperuser(ctx)
}

func (TenantService) DeleteBefore(ctx *types.ServiceContext, _ *modeliamtenant.Tenant) error {
	return ensureTenantModuleSuperuser(ctx)
}

func (TenantService) UpdateBefore(ctx *types.ServiceContext, _ *modeliamtenant.Tenant) error {
	return ensureTenantModuleSuperuser(ctx)
}

func (TenantService) PatchBefore(ctx *types.ServiceContext, _ *modeliamtenant.Tenant) error {
	return ensureTenantModuleSuperuser(ctx)
}

func (TenantService) ListBefore(ctx *types.ServiceContext, _ *[]*modeliamtenant.Tenant) error {
	return ensureTenantModuleSuperuser(ctx)
}

func (TenantService) GetBefore(ctx *types.ServiceContext, _ *modeliamtenant.Tenant) error {
	return ensureTenantModuleSuperuser(ctx)
}

func (TenantService) CreateManyBefore(ctx *types.ServiceContext, _ ...*modeliamtenant.Tenant) error {
	return ensureTenantModuleSuperuser(ctx)
}

func (TenantService) DeleteManyBefore(ctx *types.ServiceContext, _ ...*modeliamtenant.Tenant) error {
	return ensureTenantModuleSuperuser(ctx)
}

func (TenantService) UpdateManyBefore(ctx *types.ServiceContext, _ ...*modeliamtenant.Tenant) error {
	return ensureTenantModuleSuperuser(ctx)
}

func (TenantService) PatchManyBefore(ctx *types.ServiceContext, _ ...*modeliamtenant.Tenant) error {
	return ensureTenantModuleSuperuser(ctx)
}

func ensureTenantModuleSuperuser(ctx *types.ServiceContext) error {
	_, session, err := serviceiamsession.GetCurrentSession(ctx)
	if err != nil {
		return err
	}

	actor := new(modeliamuser.User)
	if err = database.Database[*modeliamuser.User](ctx.DatabaseContext()).Get(actor, session.UserID); err != nil {
		return types.NewServiceErrorWithCause(http.StatusUnauthorized, "current user not found", err)
	}
	if actor.ID == "" {
		return types.NewServiceError(http.StatusUnauthorized, "current user not found")
	}
	if actor.Username == consts.AUTHZ_USER_ROOT || actor.Username == consts.AUTHZ_USER_ADMIN {
		return nil
	}
	if actor.IsSuperuser != nil && *actor.IsSuperuser {
		return nil
	}
	return types.NewServiceError(http.StatusForbidden, "forbidden: superuser privileges required", response.CodeForbidden)
}
