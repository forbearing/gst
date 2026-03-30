package modeliam

import "github.com/forbearing/gst/model"

// EmailChangeRequest is the model for POST /api/iam/email-change-request.
// It starts a protected email change flow for the current authenticated user.
type EmailChangeRequest struct {
	model.Empty
}

// EmailChangeRequestReq is the payload for POST /api/iam/email-change-request.
// It carries the target email address and the current password for re-authentication.
type EmailChangeRequestReq struct {
	NewEmail        string `json:"new_email" validate:"required,email"`
	CurrentPassword string `json:"current_password" validate:"required"`
}

// EmailChangeRequestRsp is the response for POST /api/iam/email-change-request.
// It returns the request result message for the email change flow.
type EmailChangeRequestRsp struct {
	Msg string `json:"msg,omitempty"`
}
