package serviceiam

import (
	"github.com/cockroachdb/errors"
	"github.com/forbearing/gst/database"
	modeliam "github.com/forbearing/gst/internal/model/iam"
	"github.com/forbearing/gst/provider/redis"
	"github.com/forbearing/gst/service"
	"github.com/forbearing/gst/types"
	"github.com/forbearing/gst/types/consts"
	"golang.org/x/crypto/bcrypt"
)

type ResetPasswordService struct {
	service.Base[*modeliam.ResetPassword, *modeliam.ResetPasswordReq, *modeliam.ResetPasswordRsp]
}

func privilegedActor(actor *modeliam.User, username string) bool {
	if username == consts.AUTHZ_USER_ROOT || username == consts.AUTHZ_USER_ADMIN {
		return true
	}
	return actor.IsSuperuser != nil && *actor.IsSuperuser
}

func mayResetUserPassword(actorUsername string, actor, target *modeliam.User) error {
	if !privilegedActor(actor, actorUsername) {
		return errors.New("forbidden: superuser privileges required")
	}
	if target.IsSuperuser != nil && *target.IsSuperuser {
		if actorUsername != consts.AUTHZ_USER_ROOT && actorUsername != consts.AUTHZ_USER_ADMIN {
			return errors.New("forbidden: only root or admin may reset a superuser password")
		}
	}
	return nil
}

func (s *ResetPasswordService) Create(ctx *types.ServiceContext, req *modeliam.ResetPasswordReq) (rsp *modeliam.ResetPasswordRsp, err error) {
	log := s.WithServiceContext(ctx, ctx.GetPhase())
	log.Info("resetpassword create")

	sessionID, err := ctx.Cookie("session_id")
	if err != nil {
		return nil, errors.New("authentication required")
	}

	sessionKey := modeliam.SessionRedisKey(modeliam.SessionNamespace, sessionID)
	_, err = redis.Cache[modeliam.Session]().Get(sessionKey)
	if err != nil {
		return nil, errors.Wrap(err, "invalid session")
	}

	actors := make([]*modeliam.User, 0)
	if err = database.Database[*modeliam.User](ctx.DatabaseContext()).WithLimit(1).WithQuery(&modeliam.User{Username: ctx.Username}).List(&actors); err != nil {
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

	if err = mayResetUserPassword(ctx.Username, actor, target); err != nil {
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

	invalidateUserSessionsByUserID(req.UserID)

	log.Info("password reset successfully", "target_user_id", req.UserID, "actor", ctx.Username)
	return &modeliam.ResetPasswordRsp{Msg: "password reset successfully"}, nil
}
