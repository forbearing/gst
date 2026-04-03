package serviceiamaccount

import (
	"github.com/cockroachdb/errors"
	"github.com/forbearing/gst/database"
	modeliam "github.com/forbearing/gst/internal/model/iam"
	modeliamaccount "github.com/forbearing/gst/internal/model/iam/account"
	serviceiamsession "github.com/forbearing/gst/internal/service/iam/session"
	"github.com/forbearing/gst/model"
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

	actorUsername, actor, target, err := loadPrivilegedActorAndTarget(ctx, req.UserID)
	if err != nil {
		log.Error("failed to resolve actor or target user", err)
		return nil, err
	}

	if err = mayManageProtectedUser(actorUsername, actor, target); err != nil {
		log.Error("account status change denied", err)
		return nil, err
	}

	if target.Status == req.Status {
		// Still revoke sessions when the target state is inactive or locked so Redis cannot drift.
		if shouldInvalidateUserSessions(req.Status) {
			serviceiamsession.InvalidateUserSessions(req.UserID)
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

	if shouldInvalidateUserSessions(req.Status) {
		serviceiamsession.InvalidateUserSessions(req.UserID)
	}

	log.Info("account status updated", "target_user_id", req.UserID, "status", req.Status, "actor", actorUsername)
	return &modeliamaccount.AccountStatusRsp{Msg: "account status updated successfully"}, nil
}
