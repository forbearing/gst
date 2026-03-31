package modeliam

import "github.com/forbearing/gst/model"

// EmailPasswordResetConfirm is the model for POST /api/iam/email-password-reset-confirm.
// It completes an email-based password reset flow with the issued reset token.
type EmailPasswordResetConfirm struct {
	model.Empty
}

// EmailPasswordResetConfirmReq is the payload for POST /api/iam/email-password-reset-confirm.
// It carries the reset token and the new password from the password reset flow.
type EmailPasswordResetConfirmReq struct {
	Token       string `json:"token" validate:"required"`
	NewPassword string `json:"new_password" validate:"required,min=6"`
}

// EmailPasswordResetConfirmRsp is the response for POST /api/iam/email-password-reset-confirm.
// It indicates whether the password has been reset successfully.
type EmailPasswordResetConfirmRsp struct {
	Reset bool   `json:"reset"`
	Msg   string `json:"msg,omitempty"`
}
