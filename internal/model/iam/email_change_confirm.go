package modeliam

import "github.com/forbearing/gst/model"

// EmailChangeConfirm is the model for POST /api/iam/email-change-confirm.
// It completes the pending email change flow with the issued confirmation token.
type EmailChangeConfirm struct {
	model.Empty
}

// EmailChangeConfirmReq is the payload for POST /api/iam/email-change-confirm.
// It carries the confirmation token from the email change link or confirmation page.
type EmailChangeConfirmReq struct {
	Token string `json:"token" validate:"required"`
}

// EmailChangeConfirmRsp is the response for POST /api/iam/email-change-confirm.
// It indicates whether the pending email change has been confirmed successfully.
type EmailChangeConfirmRsp struct {
	Changed bool   `json:"changed"`
	Msg     string `json:"msg,omitempty"`
}
