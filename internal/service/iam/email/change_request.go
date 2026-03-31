package serviceiamemail

import (
	"strings"

	"github.com/cockroachdb/errors"
	"github.com/forbearing/gst/database"
	modeliam "github.com/forbearing/gst/internal/model/iam"
	modeliamemail "github.com/forbearing/gst/internal/model/iam/email"
	"github.com/forbearing/gst/service"
	"github.com/forbearing/gst/types"
	"golang.org/x/crypto/bcrypt"
)

// ChangeRequestService handles authenticated requests that start the email
// change flow for the current account.
type ChangeRequestService struct {
	service.Base[*modeliamemail.ChangeRequest, *modeliamemail.ChangeRequestReq, *modeliamemail.ChangeRequestRsp]
}

var (
	// changeLoadUserByID loads the authenticated user that initiated the change flow.
	changeLoadUserByID = func(ctx *types.ServiceContext, userID string) (*modeliam.User, error) {
		user := new(modeliam.User)
		if err := database.Database[*modeliam.User](ctx.DatabaseContext()).Get(user, userID); err != nil {
			return nil, err
		}
		return user, nil
	}
	// changeLookupUserByEmail loads the account currently bound to an email address.
	changeLookupUserByEmail = func(ctx *types.ServiceContext, email string) (*modeliam.User, error) {
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
)

// Create validates the current password, checks the target email, and issues
// one-time confirmation and cancellation tokens for the email change flow.
func (s *ChangeRequestService) Create(ctx *types.ServiceContext, req *modeliamemail.ChangeRequestReq) (rsp *modeliamemail.ChangeRequestRsp, err error) {
	log := s.WithServiceContext(ctx, ctx.GetPhase())
	user, newEmail, rsp, err := prepareEmailChangeRequest(ctx, req.NewEmail)
	if err != nil || user == nil {
		if err != nil {
			log.Error("failed to prepare email change request", err)
		}
		return rsp, err
	}

	if err = verifyEmailChangePassword(user, req.CurrentPassword); err != nil {
		log.Error("failed to verify email change password", err)
		return nil, err
	}
	if err = startEmailChangeFlow(ctx, user, newEmail, true); err != nil {
		log.Error("failed to start email change flow", err)
		return nil, err
	}

	return rsp, nil
}

// prepareEmailChangeRequest loads the current user and validates whether the new
// email can enter the change flow.
func prepareEmailChangeRequest(ctx *types.ServiceContext, newEmail string) (*modeliam.User, string, *modeliamemail.ChangeRequestRsp, error) {
	if ctx == nil || strings.TrimSpace(ctx.UserID) == "" {
		return nil, "", nil, errors.New("authentication required")
	}

	user, err := changeLoadUserByID(ctx, ctx.UserID)
	if err != nil {
		return nil, "", nil, errors.Wrap(err, "failed to load current user")
	}

	normalizedNewEmail := normalizeEmailScope(newEmail)
	if err = validateEmailChangeTarget(ctx, user, normalizedNewEmail); err != nil {
		return nil, "", nil, err
	}

	return user, normalizedNewEmail, &modeliamemail.ChangeRequestRsp{
		Msg: "email change request submitted successfully",
	}, nil
}

// validateEmailChangeTarget ensures the current account can start an email
// change flow to the requested target address.
func validateEmailChangeTarget(ctx *types.ServiceContext, user *modeliam.User, newEmail string) error {
	if user == nil || strings.TrimSpace(user.ID) == "" {
		return errors.New("current user is required")
	}
	if user.Status != "" && user.Status != modeliam.UserStatusActive {
		return errors.New("current user is not active")
	}
	currentEmail := normalizePasswordResetEmail(user.Email)
	if currentEmail == "" {
		return errors.New("current email is required")
	}
	if newEmail == "" {
		return errors.New("new email is required")
	}
	if newEmail == currentEmail {
		return errors.New("new email must be different from current email")
	}

	existingUser, err := changeLookupUserByEmail(ctx, newEmail)
	if err != nil {
		return errors.Wrap(err, "failed to lookup target email")
	}
	if existingUser != nil && existingUser.ID != user.ID {
		return errors.New("new email is already in use")
	}

	return nil
}

// verifyEmailChangePassword re-authenticates the current user before issuing
// email change tokens.
func verifyEmailChangePassword(user *modeliam.User, password string) error {
	if user == nil {
		return errors.New("current user is required")
	}
	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)); err != nil {
		return errors.New("current password is incorrect")
	}
	return nil
}

// startEmailChangeFlow issues the required tokens and dispatches the email
// change notifications for the target flow.
func startEmailChangeFlow(ctx *types.ServiceContext, user *modeliam.User, newEmail string, includeCancel bool) error {
	currentEmail := normalizePasswordResetEmail(user.Email)
	if _, err := reserveEmailThrottle(ctx.Context(), iamEmailFlowKindChangeConfirm, emailThrottleRequest, newEmail, 0); err != nil {
		if errors.Is(err, errEmailFlowThrottled) {
			return errors.Wrap(err, "email change confirmation throttled")
		}
		return errors.Wrap(err, "failed to reserve email change confirmation throttle")
	}
	if includeCancel {
		if _, err := reserveEmailThrottle(ctx.Context(), iamEmailFlowKindChangeCancel, emailThrottleRequest, currentEmail, 0); err != nil {
			if errors.Is(err, errEmailFlowThrottled) {
				return errors.Wrap(err, "email change cancellation throttled")
			}
			return errors.Wrap(err, "failed to reserve email change cancellation throttle")
		}
	}

	confirmToken, confirmFlow, err := issueEmailFlow(ctx.Context(), iamEmailFlowKindChangeConfirm, iamEmailFlowState{
		UserID:   user.ID,
		OldEmail: currentEmail,
		NewEmail: newEmail,
		Email:    newEmail,
	}, 0)
	if err != nil {
		return errors.Wrap(err, "failed to issue email change confirmation flow")
	}
	if err = dispatchEmail(ctx.Context(), changeConfirmDelivery(confirmToken, confirmFlow)); err != nil {
		return errors.Wrap(err, "failed to dispatch email change confirmation")
	}

	if !includeCancel {
		return nil
	}

	cancelToken, cancelFlow, err := issueEmailFlow(ctx.Context(), iamEmailFlowKindChangeCancel, iamEmailFlowState{
		UserID:   user.ID,
		OldEmail: currentEmail,
		NewEmail: newEmail,
		Email:    currentEmail,
	}, 0)
	if err != nil {
		return errors.Wrap(err, "failed to issue email change cancellation flow")
	}
	if err = dispatchEmail(ctx.Context(), changeCancelDelivery(cancelToken, cancelFlow)); err != nil {
		return errors.Wrap(err, "failed to dispatch email change cancellation")
	}

	return nil
}

// changeConfirmDelivery builds the email payload delivered to the new email address.
func changeConfirmDelivery(token string, flow iamEmailFlowState) emailDelivery {
	return emailDelivery{
		To:       flow.NewEmail,
		Subject:  "Email change confirmation",
		Template: "iam/email/change-confirm",
		Data: map[string]any{
			"token":      token,
			"user_id":    flow.UserID,
			"new_email":  flow.NewEmail,
			"old_email":  flow.OldEmail,
			"expires_at": flow.ExpiresAt,
		},
	}
}

// changeCancelDelivery builds the email payload delivered to the current email
// address so the user can cancel an unexpected change request.
func changeCancelDelivery(token string, flow iamEmailFlowState) emailDelivery {
	return emailDelivery{
		To:       flow.OldEmail,
		Subject:  "Email change cancellation",
		Template: "iam/email/change-cancel",
		Data: map[string]any{
			"token":      token,
			"user_id":    flow.UserID,
			"new_email":  flow.NewEmail,
			"old_email":  flow.OldEmail,
			"expires_at": flow.ExpiresAt,
		},
	}
}
