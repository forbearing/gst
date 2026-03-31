package serviceiamemail

import (
	"context"

	"github.com/cockroachdb/errors"
	"github.com/forbearing/gst/database"
	modeliam "github.com/forbearing/gst/internal/model/iam"
	modeliamemail "github.com/forbearing/gst/internal/model/iam/email"
	"github.com/forbearing/gst/service"
	"github.com/forbearing/gst/types"
)

type PasswordResetRequestService struct {
	service.Base[*modeliamemail.PasswordResetRequest, *modeliamemail.PasswordResetRequestReq, *modeliamemail.PasswordResetRequestRsp]
}

var passwordResetLookupUserByEmail = func(ctx *types.ServiceContext, email string) (*modeliam.User, error) {
	users := make([]*modeliam.User, 0, 1)
	queryEmail := email
	if err := database.Database[*modeliam.User](ctx.DatabaseContext()).
		WithLimit(1).
		WithQuery(&modeliam.User{Email: &queryEmail}).
		List(&users); err != nil {
		return nil, err
	}
	if len(users) == 0 {
		return nil, nil
	}
	return users[0], nil
}

func (s *PasswordResetRequestService) Create(ctx *types.ServiceContext, req *modeliamemail.PasswordResetRequestReq) (rsp *modeliamemail.PasswordResetRequestRsp, err error) {
	log := s.WithServiceContext(ctx, ctx.GetPhase())
	rsp = &modeliamemail.PasswordResetRequestRsp{Msg: publicAcceptedMessage(iamEmailFlowKindPasswordReset)}

	email := normalizeEmailScope(req.Email)
	if email == "" {
		return rsp, nil
	}

	if _, err = reserveEmailThrottle(ctx.Context(), iamEmailFlowKindPasswordReset, emailThrottleRequest, email, 0); err != nil {
		if errors.Is(err, errEmailFlowThrottled) {
			return rsp, nil
		}
		log.Error("failed to reserve password reset throttle", err)
		return nil, errors.Wrap(err, "failed to reserve password reset throttle")
	}

	user, err := passwordResetLookupUserByEmail(ctx, email)
	if err != nil {
		log.Error("failed to load password reset user", err)
		return nil, errors.Wrap(err, "failed to load password reset user")
	}
	if !eligiblePasswordResetUser(user, email) {
		return rsp, nil
	}

	token, flow, err := issueEmailFlow(ctx.Context(), iamEmailFlowKindPasswordReset, iamEmailFlowState{
		UserID: user.ID,
		Email:  email,
	}, 0)
	if err != nil {
		log.Error("failed to issue password reset flow", err)
		return nil, errors.Wrap(err, "failed to issue password reset flow")
	}

	if err = dispatchEmail(ctx.Context(), passwordResetDelivery(token, flow)); err != nil {
		log.Error("failed to dispatch password reset email", err)
		return nil, errors.Wrap(err, "failed to dispatch password reset email")
	}

	return rsp, nil
}

func eligiblePasswordResetUser(user *modeliam.User, email string) bool {
	if user == nil || user.ID == "" {
		return false
	}
	if normalizePasswordResetEmail(user.Email) != email {
		return false
	}
	return user.Status == "" || user.Status == modeliam.UserStatusActive
}

func passwordResetDelivery(token string, flow iamEmailFlowState) emailDelivery {
	return emailDelivery{
		To:       flow.Email,
		Subject:  "Password reset",
		Template: "iam/email/password-reset",
		Data: map[string]any{
			"token":      token,
			"user_id":    flow.UserID,
			"email":      flow.Email,
			"expires_at": flow.ExpiresAt,
		},
	}
}

func normalizePasswordResetEmail(email *string) string {
	if email == nil {
		return ""
	}
	return normalizeEmailScope(*email)
}

func passwordResetContext(ctx *types.ServiceContext) context.Context {
	if ctx == nil {
		return context.Background()
	}
	return ctx.Context()
}
