package serviceiam

import (
	modeliam "github.com/forbearing/gst/internal/model/iam"
	"github.com/forbearing/gst/service"
)

type EmailVerificationResendService struct {
	service.Base[*modeliam.EmailVerificationResend, *modeliam.EmailVerificationResendReq, *modeliam.EmailVerificationResendRsp]
}
