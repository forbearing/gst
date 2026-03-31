package serviceiam

import (
	modeliam "github.com/forbearing/gst/internal/model/iam"
	"github.com/forbearing/gst/service"
)

type EmailPasswordResetConfirmService struct {
	service.Base[*modeliam.EmailPasswordResetConfirm, *modeliam.EmailPasswordResetConfirmReq, *modeliam.EmailPasswordResetConfirmRsp]
}
