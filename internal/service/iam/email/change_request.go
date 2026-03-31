package serviceiamemail

import (
	modeliamemail "github.com/forbearing/gst/internal/model/iam/email"
	"github.com/forbearing/gst/service"
)

type ChangeRequestService struct {
	service.Base[*modeliamemail.ChangeRequest, *modeliamemail.ChangeRequestReq, *modeliamemail.ChangeRequestRsp]
}
