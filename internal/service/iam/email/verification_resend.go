package serviceiamemail

import (
	modeliamemail "github.com/forbearing/gst/internal/model/iam/email"
	"github.com/forbearing/gst/service"
)

type VerificationResendService struct {
	service.Base[*modeliamemail.VerificationResend, *modeliamemail.VerificationResendReq, *modeliamemail.VerificationResendRsp]
}
