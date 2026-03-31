package serviceiam

import (
	modeliam "github.com/forbearing/gst/internal/model/iam"
	"github.com/forbearing/gst/service"
)

type EmailChangeCancelService struct {
	service.Base[*modeliam.EmailChangeCancel, *modeliam.EmailChangeCancelReq, *modeliam.EmailChangeCancelRsp]
}
