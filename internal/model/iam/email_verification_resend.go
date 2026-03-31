package modeliam

import "github.com/forbearing/gst/model"

// EmailVerificationResend is the model for POST /api/iam/email-verification-resend.
// It resends a verification message for an email that is still waiting to be verified.
type EmailVerificationResend struct {
	model.Empty
}

// EmailVerificationResendReq is the payload for POST /api/iam/email-verification-resend.
// It identifies the email address that should receive a new verification message.
type EmailVerificationResendReq struct {
	Email string `json:"email" validate:"required,email"`
}

// EmailVerificationResendRsp is the response for POST /api/iam/email-verification-resend.
// It returns the resend result message for the verification flow.
type EmailVerificationResendRsp struct {
	Msg string `json:"msg,omitempty"`
}
