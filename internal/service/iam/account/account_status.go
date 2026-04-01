package serviceiamaccount

import (
	"github.com/cockroachdb/errors"
	"github.com/forbearing/gst/database"
	modeliam "github.com/forbearing/gst/internal/model/iam"
	modeliamaccount "github.com/forbearing/gst/internal/model/iam/account"
	modeliamsession "github.com/forbearing/gst/internal/model/iam/session"
	serviceiamsession "github.com/forbearing/gst/internal/service/iam/session"
	"github.com/forbearing/gst/model"
	"github.com/forbearing/gst/provider/redis"
	"github.com/forbearing/gst/service"
	"github.com/forbearing/gst/types"
)

type AccountStatusService struct {
	service.Base[*model.Empty, *modeliamaccount.AccountStatusReq, *modeliamaccount.AccountStatusRsp]
}

func (s *AccountStatusService) Create(ctx *types.ServiceContext, req *modeliamaccount.AccountStatusReq) (rsp *modeliamaccount.AccountStatusRsp, err error) {
	log := s.WithServiceContext(ctx, ctx.GetPhase())
	log.Info("account status create")

	if req.UserID == "" {
		return nil, errors.New("user_id is required")
	}
	switch req.Status {
	case modeliam.UserStatusActive, modeliam.UserStatusInactive, modeliam.UserStatusLocked:
	default:
		return nil, errors.New("invalid status: must be active, inactive, or locked")
	}

	sessionID, err := ctx.Cookie("session_id")
	if err != nil {
		return nil, errors.New("authentication required")
	}

	sessionKey := modeliam.SessionRedisKey(modeliam.SessionNamespace, sessionID)
	session, err := redis.Cache[modeliamsession.Session]().Get(sessionKey)
	if err != nil {
		return nil, errors.Wrap(err, "invalid session")
	}
	actorUsername := session.Username
	if actorUsername == "" {
		actorUsername = ctx.Username
	}
	if actorUsername == "" {
		return nil, errors.New("actor username not found")
	}

	actors := make([]*modeliam.User, 0)
	if err = database.Database[*modeliam.User](ctx.DatabaseContext()).WithLimit(1).WithQuery(&modeliam.User{Username: actorUsername}).List(&actors); err != nil {
		log.Error("failed to query actor user", err)
		return nil, errors.Wrap(err, "database error")
	}
	if len(actors) == 0 {
		return nil, errors.New("actor user not found")
	}
	actor := actors[0]

	target := new(modeliam.User)
	if err = database.Database[*modeliam.User](ctx.DatabaseContext()).Get(target, req.UserID); err != nil {
		log.Error("failed to load target user", err)
		return nil, errors.Wrap(err, "user not found")
	}

	if err = mayManageProtectedUser(actorUsername, actor, target); err != nil {
		log.Error("account status change denied", err)
		return nil, err
	}

	if target.Status == req.Status {
		// Still revoke sessions when the target state is inactive or locked so Redis cannot drift.
		if req.Status == modeliam.UserStatusInactive || req.Status == modeliam.UserStatusLocked {
			serviceiamsession.InvalidateUserSessionsByUserID(req.UserID)
		}
		return &modeliamaccount.AccountStatusRsp{Msg: "account status unchanged"}, nil
	}

	target.Status = req.Status
	if err = database.Database[*modeliam.User](ctx.DatabaseContext()).
		WithoutHook().
		WithSelect("username", "status").
		Update(target); err != nil {
		log.Error("failed to update user status", err)
		return nil, errors.Wrap(err, "failed to update account status")
	}

	if req.Status == modeliam.UserStatusInactive || req.Status == modeliam.UserStatusLocked {
		serviceiamsession.InvalidateUserSessionsByUserID(req.UserID)
	}

	log.Info("account status updated", "target_user_id", req.UserID, "status", req.Status, "actor", actorUsername)
	return &modeliamaccount.AccountStatusRsp{Msg: "account status updated successfully"}, nil
}
