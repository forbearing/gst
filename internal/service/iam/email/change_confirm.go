package serviceiamemail

import (
	modeliamemail "github.com/forbearing/gst/internal/model/iam/email"
	"github.com/forbearing/gst/service"
)

type ChangeConfirmService struct {
	service.Base[*modeliamemail.ChangeConfirm, *modeliamemail.ChangeConfirmReq, *modeliamemail.ChangeConfirmRsp]
}
