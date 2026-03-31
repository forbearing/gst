package serviceiam

import (
	modeliam "github.com/forbearing/gst/internal/model/iam"
	"github.com/forbearing/gst/service"
)

type EmailPasswordResetRequestService struct {
	service.Base[*modeliam.EmailPasswordResetRequest, *modeliam.EmailPasswordResetRequestReq, *modeliam.EmailPasswordResetRequestRsp]
}
