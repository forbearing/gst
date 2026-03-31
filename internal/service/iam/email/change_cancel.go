package serviceiamemail

import (
	modeliamemail "github.com/forbearing/gst/internal/model/iam/email"
	"github.com/forbearing/gst/service"
)

type ChangeCancelService struct {
	service.Base[*modeliamemail.ChangeCancel, *modeliamemail.ChangeCancelReq, *modeliamemail.ChangeCancelRsp]
}
