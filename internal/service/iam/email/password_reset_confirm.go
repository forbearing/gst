package serviceiamemail

import (
	modeliamemail "github.com/forbearing/gst/internal/model/iam/email"
	"github.com/forbearing/gst/service"
)

type PasswordResetConfirmService struct {
	service.Base[*modeliamemail.PasswordResetConfirm, *modeliamemail.PasswordResetConfirmReq, *modeliamemail.PasswordResetConfirmRsp]
}
