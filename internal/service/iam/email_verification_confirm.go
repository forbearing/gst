package serviceiam

import (
	modeliam "github.com/forbearing/gst/internal/model/iam"
	"github.com/forbearing/gst/service"
)

type EmailVerificationConfirmService struct {
	service.Base[*modeliam.EmailVerificationConfirm, *modeliam.EmailVerificationConfirmReq, *modeliam.EmailVerificationConfirmRsp]
}
