package serviceiamemail

import (
	"github.com/cockroachdb/errors"
	modeliamemail "github.com/forbearing/gst/internal/model/iam/email"
	"github.com/forbearing/gst/model"
	"github.com/forbearing/gst/service"
	"github.com/forbearing/gst/types"
)

// VerificationResendService handles public requests that resend verification
// emails for accounts that are still pending verification.
type VerificationResendService struct {
	service.Base[*model.Empty, *modeliamemail.VerificationResendReq, *modeliamemail.VerificationResendRsp]
}

// Create resends a verification email for an eligible account while keeping the
// response stable for unknown or already verified accounts.
func (s *VerificationResendService) Create(ctx *types.ServiceContext, req *modeliamemail.VerificationResendReq) (rsp *modeliamemail.VerificationResendRsp, err error) {
	log := s.WithServiceContext(ctx, ctx.GetPhase())
	rsp = &modeliamemail.VerificationResendRsp{Msg: publicAcceptedMessage(iamEmailFlowKindVerification)}

	email := normalizeEmailScope(req.Email)
	if email == "" {
		return rsp, nil
	}

	if _, err = reserveEmailThrottle(ctx.Context(), iamEmailFlowKindVerification, emailThrottleResend, email, 0); err != nil {
		if errors.Is(err, errEmailFlowThrottled) {
			return rsp, nil
		}
		log.Error("failed to reserve verification resend throttle", err)
		return nil, errors.Wrap(err, "failed to reserve verification resend throttle")
	}

	user, err := verificationLookupUserByEmail(ctx, email)
	if err != nil {
		log.Error("failed to load verification resend user", err)
		return nil, errors.Wrap(err, "failed to load verification resend user")
	}
	if !eligibleVerificationUser(user, email) {
		return rsp, nil
	}

	token, flow, err := issueEmailFlow(ctx.Context(), iamEmailFlowKindVerification, iamEmailFlowState{
		UserID: user.ID,
		Email:  email,
	}, 0)
	if err != nil {
		log.Error("failed to issue verification resend flow", err)
		return nil, errors.Wrap(err, "failed to issue verification resend flow")
	}

	if err = dispatchEmail(ctx.Context(), verificationDelivery(token, flow)); err != nil {
		log.Error("failed to dispatch verification resend email", err)
		return nil, errors.Wrap(err, "failed to dispatch verification resend email")
	}

	return rsp, nil
}
