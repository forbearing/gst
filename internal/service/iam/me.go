package serviceiam

import (
	"net/http"

	modeliam "github.com/forbearing/gst/internal/model/iam"
	"github.com/forbearing/gst/provider/redis"
	"github.com/forbearing/gst/service"
	"github.com/forbearing/gst/types"
)

type MeService struct {
	service.Base[*modeliam.Me, *modeliam.Me, modeliam.MeRsp]
}

func (s *MeService) List(ctx *types.ServiceContext, req *modeliam.Me) (rsp modeliam.MeRsp, err error) {
	log := s.WithServiceContext(ctx, ctx.GetPhase())

	sessionID, err := ctx.Cookie("session_id")
	if err != nil {
		log.Error(err)
		return nil, types.NewServiceError(http.StatusUnauthorized, err.Error())
	}

	session, e := redis.Cache[modeliam.Session]().Get(modeliam.SessionRedisKey(modeliam.SessionNamespace, sessionID))
	if e != nil {
		log.Error("session not exists")
		return nil, types.NewServiceErrorWithCause(http.StatusUnauthorized, "session not exists", e)
	}

	rsp = session.UserInfo

	return rsp, nil
}
