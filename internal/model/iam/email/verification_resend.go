package modeliamemail

import "github.com/forbearing/gst/model"

// VerificationResend is the model for POST /api/iam/email/verification-resend.
// It resends a verification message for an email that is still waiting to be verified.
type VerificationResend struct {
	model.Empty
}

// VerificationResendReq is the payload for POST /api/iam/email/verification-resend.
// It identifies the email address that should receive a new verification message.
type VerificationResendReq struct {
	Email string `json:"email" validate:"required,email"`
}

// VerificationResendRsp is the response for POST /api/iam/email/verification-resend.
// It returns the resend result message for the verification flow.
type VerificationResendRsp struct {
	Msg string `json:"msg,omitempty"`
}
