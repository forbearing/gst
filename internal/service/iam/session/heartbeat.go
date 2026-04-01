package serviceiamsession

import (
	"fmt"

	"github.com/forbearing/gst/database"
	modeliamsession "github.com/forbearing/gst/internal/model/iam/session"
	"github.com/forbearing/gst/service"
	"github.com/forbearing/gst/types"
	"github.com/mssola/useragent"
)

type HeartbeatService struct {
	service.Base[*modeliamsession.Heartbeat, *modeliamsession.Heartbeat, *modeliamsession.Heartbeat]
}

func (s *HeartbeatService) Create(ctx *types.ServiceContext, req *modeliamsession.Heartbeat) (rsp *modeliamsession.Heartbeat, err error) {
	log := s.WithServiceContext(ctx, ctx.GetPhase())

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

	// Return a non-nil response so response logging (zap.ObjectMarshaler) does not panic on nil receiver.
	return &modeliamsession.Heartbeat{}, nil
}
