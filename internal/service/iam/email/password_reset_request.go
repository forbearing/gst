package serviceiamemail

import (
	modeliamemail "github.com/forbearing/gst/internal/model/iam/email"
	"github.com/forbearing/gst/service"
)

type PasswordResetRequestService struct {
	service.Base[*modeliamemail.PasswordResetRequest, *modeliamemail.PasswordResetRequestReq, *modeliamemail.PasswordResetRequestRsp]
}
