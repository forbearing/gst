package serviceiamsession

import (
	modeliam "github.com/forbearing/gst/internal/model/iam"
	modeliamsession "github.com/forbearing/gst/internal/model/iam/session"
	"github.com/forbearing/gst/model"
	"github.com/forbearing/gst/provider/redis"
	"github.com/forbearing/gst/service"
	"github.com/forbearing/gst/types"
)

type OfflineService struct {
	service.Base[*model.Empty, *modeliamsession.OfflineReq, *model.Empty]
}

func (s *OfflineService) Create(ctx *types.ServiceContext, req *modeliamsession.OfflineReq) (rsp *model.Empty, err error) {
	log := s.WithServiceContext(ctx, ctx.GetPhase())

	prefixedUserID := modeliam.SessionRedisKey(modeliam.SessionNamespace, req.UserID)

	prefixedSessionID, err := redis.Cache[string]().Get(prefixedUserID)
	if err != nil {
		log.Error(err)
		return nil, err
	}

	if err = redis.Cache[modeliam.User]().Delete(prefixedSessionID); err != nil {
		log.Error(err)
		return nil, err
	}

	return rsp, nil
}
