package modeliam

import "github.com/forbearing/gst/model"

// EmailVerificationConfirm is the model for POST /api/iam/email-verification-confirm.
// It completes the email verification flow with the issued verification token.
type EmailVerificationConfirm struct {
	model.Empty
}

// EmailVerificationConfirmReq is the payload for POST /api/iam/email-verification-confirm.
// It carries the verification token from the email link or client-side confirmation flow.
type EmailVerificationConfirmReq struct {
	Token string `json:"token" validate:"required"`
}

// EmailVerificationConfirmRsp is the response for POST /api/iam/email-verification-confirm.
// It indicates whether the email has been verified successfully.
type EmailVerificationConfirmRsp struct {
	Verified bool   `json:"verified"`
	Msg      string `json:"msg,omitempty"`
}
