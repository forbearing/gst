package modeliam

import "github.com/forbearing/gst/model"

// EmailChangeResend is the model for POST /api/iam/email-change-resend.
// It resends a confirmation message for a pending email change request.
type EmailChangeResend struct {
	model.Empty
}

// EmailChangeResendReq is the payload for POST /api/iam/email-change-resend.
// It identifies the pending target email address that should receive a new confirmation message.
type EmailChangeResendReq struct {
	NewEmail string `json:"new_email" validate:"required,email"`
}

// EmailChangeResendRsp is the response for POST /api/iam/email-change-resend.
// It returns the resend result message for the email change flow.
type EmailChangeResendRsp struct {
	Msg string `json:"msg,omitempty"`
}
