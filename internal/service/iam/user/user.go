package serviceiamuser

import (
	"net/http"

	"github.com/forbearing/gst/database"
	modeliamuser "github.com/forbearing/gst/internal/model/iam/user"
	serviceiamsession "github.com/forbearing/gst/internal/service/iam/session"
	"github.com/forbearing/gst/response"
	"github.com/forbearing/gst/service"
	"github.com/forbearing/gst/types"
	"github.com/forbearing/gst/types/consts"
)

// UserService handles CRUD operations for IAM users.
type UserService struct {
	service.Base[*modeliamuser.User, *modeliamuser.User, *modeliamuser.User]
}

func (UserService) CreateBefore(ctx *types.ServiceContext, req *modeliamuser.User) error {
	actorUsername, actor, err := userResourceActor(ctx)
	if err != nil {
		return err
	}
	if err = ensureUserModuleSuperuser(actorUsername, actor); err != nil {
		return err
	}
	return ensureUserCreateAllowed(actorUsername, req)
}

func (UserService) ListBefore(ctx *types.ServiceContext, _ *[]*modeliamuser.User) error {
	actorUsername, actor, err := userResourceActor(ctx)
	if err != nil {
		return err
	}
	return ensureUserModuleSuperuser(actorUsername, actor)
}

func (UserService) GetBefore(ctx *types.ServiceContext, req *modeliamuser.User) error {
	return ensureUserTargetAccessible(ctx, req)
}

func (UserService) UpdateBefore(ctx *types.ServiceContext, req *modeliamuser.User) error {
	actorUsername, actor, err := userResourceActor(ctx)
	if err != nil {
		return err
	}
	if err = ensureUserModuleSuperuser(actorUsername, actor); err != nil {
		return err
	}
	return ensureUserMutationAllowed(actorUsername, req)
}

func (UserService) PatchBefore(ctx *types.ServiceContext, req *modeliamuser.User) error {
	actorUsername, actor, err := userResourceActor(ctx)
	if err != nil {
		return err
	}
	if err = ensureUserModuleSuperuser(actorUsername, actor); err != nil {
		return err
	}
	return ensureUserMutationAllowed(actorUsername, req)
}

func (UserService) DeleteBefore(ctx *types.ServiceContext, req *modeliamuser.User) error {
	actorUsername, actor, err := userResourceActor(ctx)
	if err != nil {
		return err
	}
	if err = ensureUserModuleSuperuser(actorUsername, actor); err != nil {
		return err
	}
	return ensureExistingUserTargetAllowed(actorUsername, req)
}

func (UserService) DeleteManyBefore(ctx *types.ServiceContext, users ...*modeliamuser.User) error {
	actorUsername, actor, err := userResourceActor(ctx)
	if err != nil {
		return err
	}
	if err = ensureUserModuleSuperuser(actorUsername, actor); err != nil {
		return err
	}
	for _, user := range users {
		if err = ensureExistingUserTargetAllowed(actorUsername, user); err != nil {
			return err
		}
	}
	return nil
}

// DeleteAfter revokes Redis sessions for the deleted user. The controller only guarantees
// M with ID set (route/query/body id); no other fields are required.
func (UserService) DeleteAfter(_ *types.ServiceContext, u *modeliamuser.User) error {
	if u == nil {
		return nil
	}
	serviceiamsession.InvalidateUserSessions(u.GetID())
	return nil
}

func (UserService) CreateManyBefore(ctx *types.ServiceContext, users ...*modeliamuser.User) error {
	actorUsername, actor, err := userResourceActor(ctx)
	if err != nil {
		return err
	}
	if err = ensureUserModuleSuperuser(actorUsername, actor); err != nil {
		return err
	}
	for _, user := range users {
		if err = ensureUserCreateAllowed(actorUsername, user); err != nil {
			return err
		}
	}
	return nil
}

// DeleteManyAfter revokes sessions for each deleted user. Items contain only IDs from the batch request.
func (UserService) DeleteManyAfter(_ *types.ServiceContext, users ...*modeliamuser.User) error {
	for _, u := range users {
		if u == nil {
			continue
		}
		serviceiamsession.InvalidateUserSessions(u.GetID())
	}
	return nil
}

func (UserService) UpdateManyBefore(ctx *types.ServiceContext, users ...*modeliamuser.User) error {
	actorUsername, actor, err := userResourceActor(ctx)
	if err != nil {
		return err
	}
	if err = ensureUserModuleSuperuser(actorUsername, actor); err != nil {
		return err
	}
	for _, user := range users {
		if err = ensureUserMutationAllowed(actorUsername, user); err != nil {
			return err
		}
	}
	return nil
}

func (UserService) PatchManyBefore(ctx *types.ServiceContext, users ...*modeliamuser.User) error {
	actorUsername, actor, err := userResourceActor(ctx)
	if err != nil {
		return err
	}
	if err = ensureUserModuleSuperuser(actorUsername, actor); err != nil {
		return err
	}
	for _, user := range users {
		if err = ensureUserMutationAllowed(actorUsername, user); err != nil {
			return err
		}
	}
	return nil
}

func userResourceActor(ctx *types.ServiceContext) (string, *modeliamuser.User, error) {
	_, session, err := serviceiamsession.GetCurrentSession(ctx)
	if err != nil {
		return "", nil, err
	}

	actor := new(modeliamuser.User)
	if err = database.Database[*modeliamuser.User](nil).Get(actor, session.UserID); err != nil {
		return "", nil, types.NewServiceErrorWithCause(http.StatusUnauthorized, "current user not found", err)
	}
	if actor.ID == "" {
		return "", nil, types.NewServiceError(http.StatusUnauthorized, "current user not found")
	}
	return actor.Username, actor, nil
}

func ensureUserModuleSuperuser(actorUsername string, actor *modeliamuser.User) error {
	if isRootOrAdmin(actorUsername) {
		return nil
	}
	if actor != nil && actor.IsSuperuser != nil && *actor.IsSuperuser {
		return nil
	}
	return types.NewServiceError(http.StatusForbidden, "forbidden: superuser privileges required", response.CodeForbidden)
}

func ensureUserTargetAccessible(ctx *types.ServiceContext, req *modeliamuser.User) error {
	actorUsername, actor, err := userResourceActor(ctx)
	if err != nil {
		return err
	}
	if err = ensureUserModuleSuperuser(actorUsername, actor); err != nil {
		return err
	}
	return ensureExistingUserTargetAllowed(actorUsername, req)
}

func ensureUserCreateAllowed(actorUsername string, req *modeliamuser.User) error {
	if req != nil && req.IsSuperuser != nil && *req.IsSuperuser && !isRootOrAdmin(actorUsername) {
		return userSuperuserTargetForbidden()
	}
	return nil
}

func ensureUserMutationAllowed(actorUsername string, req *modeliamuser.User) error {
	if err := ensureExistingUserTargetAllowed(actorUsername, req); err != nil {
		return err
	}
	return ensureUserCreateAllowed(actorUsername, req)
}

func ensureExistingUserTargetAllowed(actorUsername string, req *modeliamuser.User) error {
	if req == nil || req.GetID() == "" {
		return types.NewServiceError(http.StatusBadRequest, "user id is required")
	}
	target := new(modeliamuser.User)
	if err := database.Database[*modeliamuser.User](nil).Get(target, req.GetID()); err != nil {
		return types.NewServiceErrorWithCause(http.StatusInternalServerError, "failed to load target user", err)
	}
	if target.ID == "" {
		return types.NewServiceError(http.StatusNotFound, "user not found")
	}
	if target.IsSuperuser != nil && *target.IsSuperuser && !isRootOrAdmin(actorUsername) {
		return userSuperuserTargetForbidden()
	}
	return nil
}

func userSuperuserTargetForbidden() error {
	return types.NewServiceError(http.StatusForbidden, "forbidden: only root or admin may operate on a superuser", response.CodeForbidden)
}

func isRootOrAdmin(username string) bool {
	return username == consts.AUTHZ_USER_ROOT || username == consts.AUTHZ_USER_ADMIN
}
