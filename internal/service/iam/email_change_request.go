package serviceiam

import (
	modeliam "github.com/forbearing/gst/internal/model/iam"
	"github.com/forbearing/gst/service"
)

type EmailChangeRequestService struct {
	service.Base[*modeliam.EmailChangeRequest, *modeliam.EmailChangeRequestReq, *modeliam.EmailChangeRequestRsp]
}
