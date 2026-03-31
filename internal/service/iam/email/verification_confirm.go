package serviceiamemail

import (
	modeliamemail "github.com/forbearing/gst/internal/model/iam/email"
	"github.com/forbearing/gst/service"
)

type VerificationConfirmService struct {
	service.Base[*modeliamemail.VerificationConfirm, *modeliamemail.VerificationConfirmReq, *modeliamemail.VerificationConfirmRsp]
}
