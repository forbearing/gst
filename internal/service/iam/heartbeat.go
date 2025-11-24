package serviceiam

import (
	"fmt"

	"github.com/forbearing/gst/database"
	modeliam "github.com/forbearing/gst/internal/model/iam"
	"github.com/forbearing/gst/service"
	"github.com/forbearing/gst/types"
	"github.com/mssola/useragent"
)

type HeartbeatService struct {
	service.Base[*modeliam.Heartbeat, *modeliam.Heartbeat, *modeliam.Heartbeat]
}

func (s *HeartbeatService) Create(ctx *types.ServiceContext, req *modeliam.Heartbeat) (rsp *modeliam.Heartbeat, err error) {
	log := s.WithServiceContext(ctx, ctx.GetPhase())

	ua := useragent.New(ctx.Request.UserAgent())
	engineName, engineVersion := ua.Engine()
	browserName, browserVersion := ua.Browser()

	if err = database.Database[*modeliam.OnlineUser](ctx.DatabaseContext()).Update(&modeliam.OnlineUser{
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

	return rsp, nil
}
