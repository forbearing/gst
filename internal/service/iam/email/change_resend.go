package serviceiamemail

import (
	modeliamemail "github.com/forbearing/gst/internal/model/iam/email"
	"github.com/forbearing/gst/service"
)

type ChangeResendService struct {
	service.Base[*modeliamemail.ChangeResend, *modeliamemail.ChangeResendReq, *modeliamemail.ChangeResendRsp]
}
