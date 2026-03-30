package serviceiam

import (
	modeliam "github.com/forbearing/gst/internal/model/iam"
	"github.com/forbearing/gst/service"
)

type EmailChangeResendService struct {
	service.Base[*modeliam.EmailChangeResend, *modeliam.EmailChangeResendReq, *modeliam.EmailChangeResendRsp]
}
