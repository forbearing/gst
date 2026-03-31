package serviceiam

import (
	modeliam "github.com/forbearing/gst/internal/model/iam"
	"github.com/forbearing/gst/service"
)

type EmailVerificationRequestService struct {
	service.Base[*modeliam.EmailVerificationRequest, *modeliam.EmailVerificationRequestReq, *modeliam.EmailVerificationRequestRsp]
}
