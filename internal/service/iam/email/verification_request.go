package serviceiamemail

import (
	"github.com/cockroachdb/errors"
	"github.com/forbearing/gst/database"
	modeliam "github.com/forbearing/gst/internal/model/iam"
	modeliamemail "github.com/forbearing/gst/internal/model/iam/email"
	"github.com/forbearing/gst/service"
	"github.com/forbearing/gst/types"
)

// VerificationRequestService handles public requests that start the email
// verification flow for an eligible user account.
type VerificationRequestService struct {
	service.Base[*modeliamemail.VerificationRequest, *modeliamemail.VerificationRequestReq, *modeliamemail.VerificationRequestRsp]
}

// verificationLookupUserByEmail resolves the user bound to the requested email.
// Tests can replace this function to avoid database fixtures.
var verificationLookupUserByEmail = func(ctx *types.ServiceContext, email string) (*modeliam.User, error) {
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

// Create starts an email verification flow and returns a generic acceptance
// message so callers cannot infer whether the target account exists.
func (s *VerificationRequestService) Create(ctx *types.ServiceContext, req *modeliamemail.VerificationRequestReq) (rsp *modeliamemail.VerificationRequestRsp, err error) {
	log := s.WithServiceContext(ctx, ctx.GetPhase())
	rsp = &modeliamemail.VerificationRequestRsp{Msg: publicAcceptedMessage(iamEmailFlowKindVerification)}

	email := normalizeEmailScope(req.Email)
	if email == "" {
		return rsp, nil
	}

	if _, err = reserveEmailThrottle(ctx.Context(), iamEmailFlowKindVerification, emailThrottleRequest, email, 0); err != nil {
		if errors.Is(err, errEmailFlowThrottled) {
			return rsp, nil
		}
		log.Error("failed to reserve verification request throttle", err)
		return nil, errors.Wrap(err, "failed to reserve verification request throttle")
	}

	user, err := verificationLookupUserByEmail(ctx, email)
	if err != nil {
		log.Error("failed to load verification user", err)
		return nil, errors.Wrap(err, "failed to load verification user")
	}
	if !eligibleVerificationUser(user, email) {
		return rsp, nil
	}

	token, flow, err := issueEmailFlow(ctx.Context(), iamEmailFlowKindVerification, iamEmailFlowState{
		UserID: user.ID,
		Email:  email,
	}, 0)
	if err != nil {
		log.Error("failed to issue verification flow", err)
		return nil, errors.Wrap(err, "failed to issue verification flow")
	}

	if err = dispatchEmail(ctx.Context(), verificationDelivery(token, flow)); err != nil {
		log.Error("failed to dispatch verification email", err)
		return nil, errors.Wrap(err, "failed to dispatch verification email")
	}

	return rsp, nil
}

// eligibleVerificationUser ensures the verification flow is only sent to an
// active account whose current email still matches the normalized request email
// and has not already been verified.
func eligibleVerificationUser(user *modeliam.User, email string) bool {
	if user == nil || user.ID == "" {
		return false
	}
	if normalizePasswordResetEmail(user.Email) != email {
		return false
	}
	if user.Status != "" && user.Status != modeliam.UserStatusActive {
		return false
	}
	return !userEmailVerified(user)
}

// verificationDelivery builds the email payload for the verification sender.
func verificationDelivery(token string, flow iamEmailFlowState) emailDelivery {
	return emailDelivery{
		To:       flow.Email,
		Subject:  "Email verification",
		Template: "iam/email/verification",
		Data: map[string]any{
			"token":      token,
			"user_id":    flow.UserID,
			"email":      flow.Email,
			"expires_at": flow.ExpiresAt,
		},
	}
}

// userEmailVerified safely returns the email verification flag for the user.
func userEmailVerified(user *modeliam.User) bool {
	return user != nil && user.EmailVerified != nil && *user.EmailVerified
}
