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
	"github.com/forbearing/gst/types/consts"
	"golang.org/x/crypto/bcrypt"
)

type ResetPasswordService struct {
	service.Base[*model.Empty, *modeliamaccount.ResetPasswordReq, *modeliamaccount.ResetPasswordRsp]
}

func privilegedActor(actor *modeliam.User, username string) bool {
	if username == consts.AUTHZ_USER_ROOT || username == consts.AUTHZ_USER_ADMIN {
		return true
	}
	return actor.IsSuperuser != nil && *actor.IsSuperuser
}

// mayManageProtectedUser allows privileged actors to act on another user; superuser targets require root or admin.
func mayManageProtectedUser(actorUsername string, actor, target *modeliam.User) error {
	if !privilegedActor(actor, actorUsername) {
		return errors.New("forbidden: superuser privileges required")
	}
	if target.IsSuperuser != nil && *target.IsSuperuser {
		if actorUsername != consts.AUTHZ_USER_ROOT && actorUsername != consts.AUTHZ_USER_ADMIN {
			return errors.New("forbidden: only root or admin may modify a superuser")
		}
	}
	return nil
}

func mayResetUserPassword(actorUsername string, actor, target *modeliam.User) error {
	return mayManageProtectedUser(actorUsername, actor, target)
}

func (s *ResetPasswordService) Create(ctx *types.ServiceContext, req *modeliamaccount.ResetPasswordReq) (rsp *modeliamaccount.ResetPasswordRsp, err error) {
	log := s.WithServiceContext(ctx, ctx.GetPhase())
	log.Info("resetpassword create")

	_, session, err := serviceiamsession.GetCurrentSession(ctx)
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

	if err = mayResetUserPassword(actorUsername, actor, target); err != nil {
		log.Error("reset password denied", err)
		return nil, err
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.NewPassword), bcrypt.DefaultCost)
	if err != nil {
		log.Error("failed to hash new password", err)
		return nil, errors.Wrap(err, "failed to process new password")
	}

	target.PasswordHash = string(hashedPassword)
	target.MustChangePassword = true
	if err = database.Database[*modeliam.User](ctx.DatabaseContext()).
		WithoutHook().
		WithSelect("username", "password_hash", "must_change_password").
		Update(target); err != nil {
		log.Error("failed to update user password fields", err)
		return nil, errors.Wrap(err, "failed to update password")
	}

	serviceiamsession.InvalidateUserSessions(req.UserID)

	log.Info("password reset successfully", "target_user_id", req.UserID, "actor", actorUsername)
	return &modeliamaccount.ResetPasswordRsp{Msg: "password reset successfully"}, nil
}
