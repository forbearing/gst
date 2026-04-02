package serviceiamsession

import (
	modeliamsession "github.com/forbearing/gst/internal/model/iam/session"
	"github.com/forbearing/gst/model"
	"github.com/forbearing/gst/service"
	"github.com/forbearing/gst/types"
)

type OfflineService struct {
	service.Base[*model.Empty, *modeliamsession.OfflineReq, *model.Empty]
}

func (s *OfflineService) Create(ctx *types.ServiceContext, req *modeliamsession.OfflineReq) (rsp *model.Empty, err error) {
	if req.UserID == "" {
		return &model.Empty{}, nil
	}

	InvalidateUserSessions(req.UserID)

	return &model.Empty{}, nil
}
