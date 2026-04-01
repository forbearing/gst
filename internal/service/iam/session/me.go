package serviceiamsession

import (
	"maps"
	"net/http"

	"github.com/forbearing/gst/database"
	modeliam "github.com/forbearing/gst/internal/model/iam"
	modeliamsession "github.com/forbearing/gst/internal/model/iam/session"
	"github.com/forbearing/gst/model"
	"github.com/forbearing/gst/provider/redis"
	"github.com/forbearing/gst/response"
	"github.com/forbearing/gst/service"
	"github.com/forbearing/gst/types"
)

type MeService struct {
	service.Base[*model.Empty, *model.Empty, modeliamsession.MeRsp]
}

func (s *MeService) List(ctx *types.ServiceContext, req *model.Empty) (rsp modeliamsession.MeRsp, err error) {
	log := s.WithServiceContext(ctx, ctx.GetPhase())

	sessionID, err := ctx.Cookie("session_id")
	if err != nil {
		log.Error(err)
		return nil, types.NewServiceError(http.StatusUnauthorized, err.Error())
	}

	session, e := redis.Cache[modeliamsession.Session]().Get(modeliamsession.SessionRedisKey(modeliamsession.SessionNamespace, sessionID))
	if e != nil {
		log.Error("session not exists")
		return nil, types.NewServiceErrorWithCause(http.StatusUnauthorized, "session not exists", e)
	}

	// Load current status from DB (single query per /me); IAMSession does not query users.
	var statusStr string
	if session.UserID != "" {
		u := new(modeliam.User)
		if err := database.Database[*modeliam.User](ctx.DatabaseContext()).Get(u, session.UserID); err != nil || u.GetID() == "" {
			log.Error("failed to load user for /me")
			return nil, types.NewServiceError(http.StatusUnauthorized, "session invalid")
		}
		switch u.Status {
		case modeliam.UserStatusInactive:
			return nil, types.NewServiceError(http.StatusForbidden, "", response.CodeAccountInactive)
		case modeliam.UserStatusLocked:
			return nil, types.NewServiceError(http.StatusForbidden, "", response.CodeAccountLocked)
		}
		statusStr = string(u.Status)
	}

	out := make(map[string]any)
	if session.UserInfo != nil {
		maps.Copy(out, session.UserInfo)
	}
	if statusStr != "" {
		out["status"] = statusStr
	}

	return out, nil
}
