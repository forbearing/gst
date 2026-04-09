package serviceiamgroup

import (
	"net/http"

	"github.com/forbearing/gst/database"
	modeliamgroup "github.com/forbearing/gst/internal/model/iam/group"
	modeliamuser "github.com/forbearing/gst/internal/model/iam/user"
	serviceiamsession "github.com/forbearing/gst/internal/service/iam/session"
	"github.com/forbearing/gst/response"
	"github.com/forbearing/gst/service"
	"github.com/forbearing/gst/types"
	"github.com/forbearing/gst/types/consts"
)

// GroupService handles CRUD operations for IAM groups.
type GroupService struct {
	service.Base[*modeliamgroup.Group, *modeliamgroup.Group, *modeliamgroup.Group]
}

func (GroupService) CreateBefore(ctx *types.ServiceContext, _ *modeliamgroup.Group) error {
	return ensureGroupModuleSuperuser(ctx)
}

func (GroupService) DeleteBefore(ctx *types.ServiceContext, _ *modeliamgroup.Group) error {
	return ensureGroupModuleSuperuser(ctx)
}

func (GroupService) UpdateBefore(ctx *types.ServiceContext, _ *modeliamgroup.Group) error {
	return ensureGroupModuleSuperuser(ctx)
}

func (GroupService) PatchBefore(ctx *types.ServiceContext, _ *modeliamgroup.Group) error {
	return ensureGroupModuleSuperuser(ctx)
}

func (GroupService) ListBefore(ctx *types.ServiceContext, _ *[]*modeliamgroup.Group) error {
	return ensureGroupModuleSuperuser(ctx)
}

func (GroupService) GetBefore(ctx *types.ServiceContext, _ *modeliamgroup.Group) error {
	return ensureGroupModuleSuperuser(ctx)
}

func (GroupService) CreateManyBefore(ctx *types.ServiceContext, _ ...*modeliamgroup.Group) error {
	return ensureGroupModuleSuperuser(ctx)
}

func (GroupService) DeleteManyBefore(ctx *types.ServiceContext, _ ...*modeliamgroup.Group) error {
	return ensureGroupModuleSuperuser(ctx)
}

func (GroupService) UpdateManyBefore(ctx *types.ServiceContext, _ ...*modeliamgroup.Group) error {
	return ensureGroupModuleSuperuser(ctx)
}

func (GroupService) PatchManyBefore(ctx *types.ServiceContext, _ ...*modeliamgroup.Group) error {
	return ensureGroupModuleSuperuser(ctx)
}

func ensureGroupModuleSuperuser(ctx *types.ServiceContext) error {
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
