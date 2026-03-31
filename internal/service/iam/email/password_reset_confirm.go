package serviceiamemail

import (
	"strings"

	"github.com/cockroachdb/errors"
	"github.com/forbearing/gst/database"
	modeliam "github.com/forbearing/gst/internal/model/iam"
	modeliamemail "github.com/forbearing/gst/internal/model/iam/email"
	"github.com/forbearing/gst/provider/redis"
	"github.com/forbearing/gst/service"
	"github.com/forbearing/gst/types"
	"golang.org/x/crypto/bcrypt"
)

type PasswordResetConfirmService struct {
	service.Base[*modeliamemail.PasswordResetConfirm, *modeliamemail.PasswordResetConfirmReq, *modeliamemail.PasswordResetConfirmRsp]
}

var (
	passwordResetLoadUserByID = func(ctx *types.ServiceContext, userID string) (*modeliam.User, error) {
		user := new(modeliam.User)
		if err := database.Database[*modeliam.User](ctx.DatabaseContext()).Get(user, userID); err != nil {
			return nil, err
		}
		return user, nil
	}
	passwordResetUpdateUser = func(ctx *types.ServiceContext, user *modeliam.User) error {
		return database.Database[*modeliam.User](ctx.DatabaseContext()).
			WithoutHook().
			WithSelect("username", "password_hash", "must_change_password").
			Update(user)
	}
	passwordResetInvalidateSessions = func(userID string) {
		if userID == "" {
			return
		}
		prefixedUserID := modeliam.SessionRedisKey(modeliam.SessionNamespace, userID)
		sessionKey, err := redis.Cache[string]().Get(prefixedUserID)
		if err == nil && sessionKey != "" {
			_ = redis.Cache[modeliam.Session]().Delete(sessionKey)
		}
		_ = redis.Cache[string]().Delete(prefixedUserID)
	}
)

func (s *PasswordResetConfirmService) Create(ctx *types.ServiceContext, req *modeliamemail.PasswordResetConfirmReq) (rsp *modeliamemail.PasswordResetConfirmRsp, err error) {
	log := s.WithServiceContext(ctx, ctx.GetPhase())

	flow, err := consumeEmailFlow(passwordResetContext(ctx), iamEmailFlowKindPasswordReset, req.Token)
	if err != nil {
		if errors.Is(err, errEmailFlowNotFound) || errors.Is(err, errEmailFlowExpired) {
			return &modeliamemail.PasswordResetConfirmRsp{
				Reset: false,
				Msg:   "invalid or expired password reset token",
			}, nil
		}
		log.Error("failed to consume password reset flow", err)
		return nil, errors.Wrap(err, "failed to consume password reset flow")
	}
	if strings.TrimSpace(flow.UserID) == "" {
		return nil, errors.New("password reset user id is required")
	}

	user, err := passwordResetLoadUserByID(ctx, flow.UserID)
	if err != nil {
		log.Error("failed to load password reset user", err)
		return nil, errors.Wrap(err, "failed to load password reset user")
	}
	if normalizePasswordResetEmail(user.Email) != normalizeEmailScope(flow.Email) {
		return &modeliamemail.PasswordResetConfirmRsp{
			Reset: false,
			Msg:   "invalid or expired password reset token",
		}, nil
	}

	if err = applyPasswordReset(user, req.NewPassword); err != nil {
		log.Error("failed to apply password reset", err)
		return nil, err
	}
	if err = passwordResetUpdateUser(ctx, user); err != nil {
		log.Error("failed to update password reset user", err)
		return nil, errors.Wrap(err, "failed to update password")
	}

	passwordResetInvalidateSessions(user.ID)
	return &modeliamemail.PasswordResetConfirmRsp{
		Reset: true,
		Msg:   "password reset successfully",
	}, nil
}

func applyPasswordReset(user *modeliam.User, newPassword string) error {
	if user == nil {
		return errors.New("password reset user is required")
	}
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
	if err != nil {
		return errors.Wrap(err, "failed to process new password")
	}
	user.PasswordHash = string(hashedPassword)
	user.MustChangePassword = false
	return nil
}
