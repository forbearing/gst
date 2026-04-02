package serviceiamsession

import (
	"fmt"
	"net/http"

	"github.com/cockroachdb/errors"
	"github.com/forbearing/gst/database"
	modeliamsession "github.com/forbearing/gst/internal/model/iam/session"
	"github.com/forbearing/gst/provider/redis"
	"github.com/forbearing/gst/service"
	"github.com/forbearing/gst/types"
	"github.com/mssola/useragent"
)

// HeartbeatService records client liveness for the current authenticated session.
type HeartbeatService struct {
	service.Base[*modeliamsession.Heartbeat, *modeliamsession.Heartbeat, *modeliamsession.Heartbeat]
}

// Create validates the current session and updates the online-user record without
// extending the Redis session lifetime.
func (s *HeartbeatService) Create(ctx *types.ServiceContext, req *modeliamsession.Heartbeat) (rsp *modeliamsession.Heartbeat, err error) {
	log := s.WithServiceContext(ctx, ctx.GetPhase())

	sessionID, err := ctx.Cookie("session_id")
	if err != nil {
		log.Error(err)
		return nil, types.NewServiceError(http.StatusUnauthorized, err.Error())
	}
	sessionKey := modeliamsession.SessionIDKey(sessionID)
	if _, err = redis.Cache[modeliamsession.Session]().Get(sessionKey); err != nil {
		log.Error("session not exists")
		return nil, types.NewServiceErrorWithCause(http.StatusUnauthorized, "session not exists", errors.Wrap(err, "invalid session"))
	}

	ua := useragent.New(ctx.Request.UserAgent())
	engineName, engineVersion := ua.Engine()
	browserName, browserVersion := ua.Browser()

	if err = database.Database[*modeliamsession.OnlineUser](ctx.DatabaseContext()).Update(&modeliamsession.OnlineUser{
		UserID:   ctx.UserID,
		ClientIP: ctx.ClientIP,
		Username: ctx.Username,
		Source:   ctx.Request.UserAgent(),
		Platform: fmt.Sprintf("%s %s", ua.Platform(), ua.OS()),
		Engine:   fmt.Sprintf("%s %s", engineName, engineVersion),
		Browser:  fmt.Sprintf("%s %s", browserName, browserVersion),
	}); err != nil {
		log.Error(err)
		return rsp, err
	}
	return &modeliamsession.Heartbeat{}, nil
}
