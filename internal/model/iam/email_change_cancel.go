package modeliam

import "github.com/forbearing/gst/model"

// EmailChangeCancel is the model for POST /api/iam/email-change-cancel.
// It cancels a pending email change flow with the issued cancellation token.
type EmailChangeCancel struct {
	model.Empty
}

// EmailChangeCancelReq is the payload for POST /api/iam/email-change-cancel.
// It carries the cancellation token from the notification sent to the old email address.
type EmailChangeCancelReq struct {
	Token string `json:"token" validate:"required"`
}

// EmailChangeCancelRsp is the response for POST /api/iam/email-change-cancel.
// It indicates whether the pending email change has been canceled successfully.
type EmailChangeCancelRsp struct {
	Canceled bool   `json:"canceled"`
	Msg      string `json:"msg,omitempty"`
}
