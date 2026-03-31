package serviceiamemail

import (
	modeliamemail "github.com/forbearing/gst/internal/model/iam/email"
	"github.com/forbearing/gst/service"
)

type VerificationRequestService struct {
	service.Base[*modeliamemail.VerificationRequest, *modeliamemail.VerificationRequestReq, *modeliamemail.VerificationRequestRsp]
}
