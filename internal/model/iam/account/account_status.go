package modeliamaccount

import modeliam "github.com/forbearing/gst/internal/model/iam"

type AccountStatusReq struct {
	UserID string              `json:"user_id" validate:"required"`
	Status modeliam.UserStatus `json:"status" validate:"required"`
}

type AccountStatusRsp struct {
	Msg string `json:"msg,omitempty"`
}
