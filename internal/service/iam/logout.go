package serviceiam

import (
	"fmt"

	"github.com/forbearing/gst/database"
	modeliam "github.com/forbearing/gst/internal/model/iam"
	"github.com/forbearing/gst/module/logmgmt"
	"github.com/forbearing/gst/provider/redis"
	"github.com/forbearing/gst/service"
	"github.com/forbearing/gst/types"
	"github.com/mssola/useragent"
	"go.uber.org/zap"
)

type LogoutService struct {
	service.Base[*modeliam.Logout, *modeliam.Logout, *modeliam.LogoutRsp]
}

func (s *LogoutService) Create(ctx *types.ServiceContext, req *modeliam.Logout) (rsp *modeliam.LogoutRsp, err error) {
	log := s.WithServiceContext(ctx, ctx.GetPhase())

	// return keycloakLogout(ctx, log, req)
	return localLogout(ctx, log, req)
}

func localLogout(ctx *types.ServiceContext, log types.Logger, req *modeliam.Logout) (rsp *modeliam.LogoutRsp, err error) {
	// Get session_id from cookie
	sessionID, err := ctx.Cookie("session_id")
	if err != nil {
		log.Error("failed to get session_id from cookie", err)
		return &modeliam.LogoutRsp{Msg: "logout successful"}, nil // Return success even if no session
	}

	// Get session from Redis to extract user info for logging
	prefixedSessionID := SessionRedisKey(modeliam.SessionNamespace, sessionID)
	session, err := redis.Cache[modeliam.Session]().Get(prefixedSessionID)

	// Parse user agent for logging
	ua := useragent.New(ctx.UserAgent)
	engineName, engineVersion := ua.Engine()
	browserName, browserVersion := ua.Browser()

	// Record logout log
	var userID, username string
	if err == nil {
		userID = session.UserID
		username = session.Username
	}

	if logErr := database.Database[*logmgmt.LoginLog](ctx.DatabaseContext()).Create(&logmgmt.LoginLog{
		UserID:   userID,
		Username: username,
		ClientIP: ctx.ClientIP,
		Status:   logmgmt.LoginStatusLogout,
		Source:   ctx.Request.UserAgent(),
		Platform: fmt.Sprintf("%s %s", ua.Platform(), ua.OS()),
		Engine:   fmt.Sprintf("%s %s", engineName, engineVersion),
		Browser:  fmt.Sprintf("%s %s", browserName, browserVersion),
	}); logErr != nil {
		log.Warnz("failed to write logout log", zap.Error(logErr))
	}

	// Delete session from Redis
	redis.Cache[modeliam.Session]().Delete(prefixedSessionID)
	// Delete session id from redis
	prefixedUserID := SessionRedisKey(modeliam.SessionNamespace, userID)
	redis.Cache[string]().Delete(prefixedUserID)

	// Clear the session cookie
	ctx.SetCookie("session_id", "", -1, "/", "", false, true)

	log.Info("user logged out successfully", "session_id", sessionID)
	return &modeliam.LogoutRsp{Msg: "logout successful"}, nil
}

// func keycloakLogout(ctx *types.ServiceContext, log types.Logger, req *iam.Logout) (rsp *iam.LogoutRsp, err error) {
// 	// 获取前端 cookie 中的 session	id
// 	sessionID, err := ctx.Cookie("session_id")
// 	if err != nil {
// 		log.Error(err)
// 		return rsp, err
// 	}
//
// 	redisKey := helper.SessionRedisKey(iam.SessionNamespace, sessionID)
// 	// 获取 redis 中的 session
// 	session, e := redis.Cache[iam.Session]().Get(redisKey)
// 	if e != nil {
// 		log.Error(e)
// 		return nil, e
// 	}
//
// 	// keycloak 中退出登录
// 	if err := keycloak.IdentityLogout(log, session.Token.RefreshToken); err != nil {
// 		log.Error(err)
// 		return nil, err
// 	}
//
// 	// 删除 redis 中的 session
// 	redis.Cache[iam.Session]().Delete(redisKey)
//
// 	return rsp, nil
// }
