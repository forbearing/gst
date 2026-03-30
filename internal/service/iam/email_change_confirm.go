package serviceiam

import (
	modeliam "github.com/forbearing/gst/internal/model/iam"
	"github.com/forbearing/gst/service"
)

type EmailChangeConfirmService struct {
	service.Base[*modeliam.EmailChangeConfirm, *modeliam.EmailChangeConfirmReq, *modeliam.EmailChangeConfirmRsp]
}
