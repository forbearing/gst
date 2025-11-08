package controller

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/forbearing/gst/config"
	"github.com/forbearing/gst/database"
	modellog "github.com/forbearing/gst/internal/model/log"
	"github.com/forbearing/gst/model"
	"github.com/forbearing/gst/provider/redis"
	"github.com/forbearing/gst/types"
	"github.com/forbearing/gst/util"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// writeCookie 写 cookie 并重定向
func writeCookie(c *gin.Context, token, userID, name string, redirect ...bool) {
	zap.S().Info("writeCookie")
	zap.S().Info("'TokenExpireDuration:' ", config.App.AccessTokenExpireDuration)
	http.SetCookie(c.Writer, &http.Cookie{
		Path:    "/",
		Name:    TOKEN,
		Value:   token,
		Expires: time.Now().Add(config.App.AccessTokenExpireDuration),
	})
	http.SetCookie(c.Writer, &http.Cookie{
		Path:    "/",
		Name:    ID,
		Value:   userID,
		Expires: time.Now().Add(config.App.AccessTokenExpireDuration),
	})
	http.SetCookie(c.Writer, &http.Cookie{
		Path:    "/",
		Name:    NAME,
		Value:   base64.StdEncoding.EncodeToString([]byte(name)), // 中文名,需要转码
		Expires: time.Now().Add(config.App.AccessTokenExpireDuration),
	})
	if len(redirect) > 0 {
		if redirect[0] {
			c.Redirect(http.StatusTemporaryRedirect, config.App.Domain)
		}
	}
}

// writeLocalSessionAndCookie
func writeLocalSessionAndCookie(c *gin.Context, aToken, rToken string, user *model.User) {
	if user == nil {
		zap.S().Info("user is nil")
		return
	}
	name := user.Name
	userID := user.ID
	sessionID := user.SessionID
	zap.S().Info("user.SessionId: ", user.SessionID)
	sessionData, err := json.Marshal(user)
	if err != nil {
		zap.S().Error(err)
		return
	}
	if err := redis.SetSession(sessionID, sessionData); err != nil {
		zap.S().Error(err)
		return
	}

	http.SetCookie(c.Writer, &http.Cookie{
		Path:    "/",
		Name:    TOKEN,
		Value:   aToken,
		Expires: time.Now().Add(config.App.AccessTokenExpireDuration),
	})
	http.SetCookie(c.Writer, &http.Cookie{
		Path:    "/",
		Name:    ACCESS_TOKEN,
		Value:   aToken,
		Expires: time.Now().Add(config.App.AccessTokenExpireDuration),
	})
	http.SetCookie(c.Writer, &http.Cookie{
		Path:  "/",
		Name:  REFRESH_TOKEN,
		Value: rToken,
		// FIXME: refresh token expire duration should defined by config.
		Expires: time.Now().Add(7 * 24 * time.Hour),
	})
	http.SetCookie(c.Writer, &http.Cookie{
		Path:    "/",
		Name:    SESSION_ID,
		Value:   sessionID,
		Expires: time.Now().Add(config.App.AccessTokenExpireDuration),
	})
	http.SetCookie(c.Writer, &http.Cookie{
		Path:    "/",
		Name:    ID,
		Value:   userID,
		Expires: time.Now().Add(config.App.AccessTokenExpireDuration),
	})
	http.SetCookie(c.Writer, &http.Cookie{
		Path:    "/",
		Name:    NAME,
		Value:   base64.StdEncoding.EncodeToString([]byte(name)), // 中文名,需要转码
		Expires: time.Now().Add(config.App.AccessTokenExpireDuration),
	})
}

// writeFeishuSessionAndCookie 写 cookie 并重定向
func writeFeishuSessionAndCookie(c *gin.Context, aToken, rToken string, userInfo *model.UserInfo) {
	if userInfo == nil {
		zap.S().Error("userInfo is nil")
		return
	}
	name := userInfo.Name
	userID := userInfo.UserID
	sessionData, err := json.Marshal(userInfo)
	if err != nil {
		zap.S().Error(err)
		return
	}
	sessionID := util.UUID()
	if err = redis.SetSession(sessionID, sessionData); err != nil {
		zap.S().Error(err)
		return
	}

	http.SetCookie(c.Writer, &http.Cookie{
		Path:    "/",
		Name:    TOKEN,
		Value:   aToken,
		Expires: time.Now().Add(config.App.AccessTokenExpireDuration),
	})
	http.SetCookie(c.Writer, &http.Cookie{
		Path:    "/",
		Name:    ACCESS_TOKEN,
		Value:   aToken,
		Expires: time.Now().Add(config.App.AccessTokenExpireDuration),
	})
	http.SetCookie(c.Writer, &http.Cookie{
		Path:  "/",
		Name:  REFRESH_TOKEN,
		Value: rToken,
		// FIXME: refresh token expire duration should defined by config.
		Expires: time.Now().Add(7 * 24 * time.Hour),
	})
	http.SetCookie(c.Writer, &http.Cookie{
		Path:    "/",
		Name:    SESSION_ID,
		Value:   sessionID,
		Expires: time.Now().Add(config.App.AccessTokenExpireDuration),
	})
	http.SetCookie(c.Writer, &http.Cookie{
		Path:    "/",
		Name:    ID,
		Value:   userID,
		Expires: time.Now().Add(config.App.AccessTokenExpireDuration),
	})
	http.SetCookie(c.Writer, &http.Cookie{
		Path:    "/",
		Name:    NAME,
		Value:   base64.StdEncoding.EncodeToString([]byte(name)), // 中文名,需要转码
		Expires: time.Now().Add(config.App.AccessTokenExpireDuration),
	})
	// ua := useragent.New(c.Request.UserAgent())
	// engineName, engineVersion := ua.Engine()
	// browserName, browserVersion := ua.Browser()
	err = database.Database[*modellog.LoginLog](types.NewDatabaseContext(c)).Create(&modellog.LoginLog{
		UserID:   userInfo.UserID,
		Username: userInfo.Name,
		// Token:    aToken,
		Status:   modellog.LoginStatusSuccess,
		ClientIP: c.ClientIP(),
		Source:   c.Request.UserAgent(),
		// UserAgent: model.UserAgent{
		// 	Source:   c.Request.UserAgent(),
		// 	Platform: fmt.Sprintf("%s %s", ua.Platform(), ua.OS()),
		// 	Engine:   fmt.Sprintf("%s %s", engineName, engineVersion),
		// 	Browser:  fmt.Sprintf("%s %s", browserName, browserVersion),
		// },
	})
	if err != nil {
		zap.S().Error(err)
	}
	domain := config.App.Domain
	if len(util.ParseScheme(c.Request)) > 0 && len(c.Request.Host) > 0 {
		domain = fmt.Sprintf("%s://%s", util.ParseScheme(c.Request), c.Request.Host)
	}
	c.Redirect(http.StatusTemporaryRedirect, domain)
}
