package serviceiam

import (
	modeliam "github.com/forbearing/gst/internal/model/iam"
	"github.com/forbearing/gst/provider/redis"
	"github.com/forbearing/gst/service"
	"github.com/forbearing/gst/types"
)

type OfflineService struct {
	service.Base[*modeliam.Offline, *modeliam.OfflineReq, *modeliam.Offline]
}

func (s *OfflineService) Create(ctx *types.ServiceContext, req *modeliam.OfflineReq) (rsp *modeliam.Offline, err error) {
	log := s.WithServiceContext(ctx, ctx.GetPhase())

	prefixedUserID := SessionRedisKey(modeliam.SessionNamespace, req.UserID)

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
