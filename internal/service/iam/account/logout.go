package serviceiamaccount

import (
	"fmt"

	"github.com/forbearing/gst/database"
	modeliamaccount "github.com/forbearing/gst/internal/model/iam/account"
	modeliamsession "github.com/forbearing/gst/internal/model/iam/session"
	modellogmgmt "github.com/forbearing/gst/internal/model/logmgmt"
	"github.com/forbearing/gst/model"
	"github.com/forbearing/gst/provider/redis"
	"github.com/forbearing/gst/service"
	"github.com/forbearing/gst/types"
	"github.com/mssola/useragent"
	"go.uber.org/zap"
)

type LogoutService struct {
	service.Base[*model.Empty, *model.Empty, *modeliamaccount.LogoutRsp]
}

func (s *LogoutService) Create(ctx *types.ServiceContext, req *model.Empty) (rsp *modeliamaccount.LogoutRsp, err error) {
	log := s.WithServiceContext(ctx, ctx.GetPhase())

	// return keycloakLogout(ctx, log, req)
	return localLogout(ctx, log, req)
}

func localLogout(ctx *types.ServiceContext, log types.Logger, req *model.Empty) (rsp *modeliamaccount.LogoutRsp, err error) {
	// Get session_id from cookie
	sessionID, err := ctx.Cookie("session_id")
	if err != nil {
		log.Error("failed to get session_id from cookie", err)
		return &modeliamaccount.LogoutRsp{Msg: "logout successful"}, nil // Return success even if no session
	}

	// Get session from Redis to extract user info for logging
	prefixedSessionID := modeliamsession.SessionRedisKey(modeliamsession.SessionNamespace, sessionID)
	session, err := redis.Cache[modeliamsession.Session]().Get(prefixedSessionID)

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

	if logErr := database.Database[*modellogmgmt.LoginLog](ctx.DatabaseContext()).Create(&modellogmgmt.LoginLog{
		UserID:   userID,
		Username: username,
		ClientIP: ctx.ClientIP,
		Status:   modellogmgmt.LoginStatusLogout,
		Source:   ctx.Request.UserAgent(),
		Platform: fmt.Sprintf("%s %s", ua.Platform(), ua.OS()),
		Engine:   fmt.Sprintf("%s %s", engineName, engineVersion),
		Browser:  fmt.Sprintf("%s %s", browserName, browserVersion),
	}); logErr != nil {
		log.Warnz("failed to write logout log", zap.Error(logErr))
	}

	// Delete session from Redis
	if delErr := redis.Cache[modeliamsession.Session]().Delete(prefixedSessionID); delErr != nil {
		log.Warnz("failed to delete session from redis", zap.Error(delErr))
	}
	// Delete session id from redis
	prefixedUserID := modeliamsession.SessionRedisKey(modeliamsession.SessionNamespace, userID)
	if delErr := redis.Cache[string]().Delete(prefixedUserID); delErr != nil {
		log.Warnz("failed to delete session id from redis", zap.Error(delErr))
	}

	// Clear the session cookie
	ctx.SetCookie("session_id", "", -1, "/", "", false, true)

	log.Info("user logged out successfully", "session_id", sessionID)
	return &modeliamaccount.LogoutRsp{Msg: "logout successful"}, nil
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
