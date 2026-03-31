package modeliam

import "github.com/forbearing/gst/model"

// EmailVerificationRequest is the model for POST /api/iam/email-verification-request.
// It starts an email verification flow for the target email address.
type EmailVerificationRequest struct {
	model.Empty
}

// EmailVerificationRequestReq is the payload for POST /api/iam/email-verification-request.
// It accepts the email address that should receive the verification message.
type EmailVerificationRequestReq struct {
	Email string `json:"email" validate:"required,email"`
}

// EmailVerificationRequestRsp is the response for POST /api/iam/email-verification-request.
// It returns the delivery result message for the verification request.
type EmailVerificationRequestRsp struct {
	Msg string `json:"msg,omitempty"`
}
