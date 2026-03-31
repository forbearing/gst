package modeliam

import "github.com/forbearing/gst/model"

// EmailPasswordResetRequest is the model for POST /api/iam/email-password-reset-request.
// It starts an email-based password reset flow for the target account.
type EmailPasswordResetRequest struct {
	model.Empty
}

// EmailPasswordResetRequestReq is the payload for POST /api/iam/email-password-reset-request.
// It accepts the email address that should receive the password reset message.
type EmailPasswordResetRequestReq struct {
	Email string `json:"email" validate:"required,email"`
}

// EmailPasswordResetRequestRsp is the response for POST /api/iam/email-password-reset-request.
// It returns the request result message for the email password reset flow.
type EmailPasswordResetRequestRsp struct {
	Msg string `json:"msg,omitempty"`
}
