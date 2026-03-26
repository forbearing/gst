package serviceiam

import (
	"time"

	"github.com/cockroachdb/errors"
	"github.com/forbearing/gst/database"
	modeliam "github.com/forbearing/gst/internal/model/iam"
	"github.com/forbearing/gst/provider/redis"
	"github.com/forbearing/gst/service"
	"github.com/forbearing/gst/types"
)

type AccountStatusService struct {
	service.Base[*modeliam.AccountStatus, *modeliam.AccountStatusReq, *modeliam.AccountStatusRsp]
}

func (s *AccountStatusService) Create(ctx *types.ServiceContext, req *modeliam.AccountStatusReq) (rsp *modeliam.AccountStatusRsp, err error) {
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
	session, err := redis.Cache[modeliam.Session]().Get(sessionKey)
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
		return &modeliam.AccountStatusRsp{Msg: "account status unchanged"}, nil
	}

	target.Status = req.Status
	now := time.Now()
	target.UpdatedAt = &now
	if err = database.Database[*modeliam.User](ctx.DatabaseContext()).
		WithoutHook().
		WithSelect("username", "status").
		Update(target); err != nil {
		log.Error("failed to update user status", err)
		return nil, errors.Wrap(err, "failed to update account status")
	}

	if req.Status == modeliam.UserStatusInactive || req.Status == modeliam.UserStatusLocked {
		invalidateUserSessionsByUserID(req.UserID)
	}

	log.Info("account status updated", "target_user_id", req.UserID, "status", req.Status, "actor", actorUsername)
	return &modeliam.AccountStatusRsp{Msg: "account status updated successfully"}, nil
}
