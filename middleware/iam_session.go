package middleware

import (
	"net/http"

	modeliam "github.com/forbearing/gst/internal/model/iam"
	"github.com/forbearing/gst/provider/redis"
	"github.com/forbearing/gst/types/consts"
	"github.com/gin-gonic/gin"
	"github.com/mssola/useragent"
)

func IAMSession() gin.HandlerFunc {
	return func(c *gin.Context) {
		// fmt.Println("----- identifySession middleware", c.Request.RequestURI)
		sessionID, err := c.Cookie("session_id")
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "no session"})
			return
		}
		// fmt.Println("----- SessionRedisKey", helper.SessionRedisKey(identity.SessionNamespace, sessionID))
		session, e := redis.Cache[modeliam.Session]().WithContext(c.Request.Context()).Get(modeliam.SessionRedisKey(modeliam.SessionNamespace, sessionID))
		if e != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": e.Error()})
			return
		}

		// 校验浏览器/OS
		ua := useragent.New(c.Request.UserAgent())
		engineName, _ := ua.Engine()
		browserName, _ := ua.Browser()
		if session.OS != ua.OS() {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "os mismatch"})
			return
		}
		if session.Platform != ua.Platform() {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "platform mismatch"})
			return
		}
		if engineName != session.EngineName {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "engine mismatch"})
			return
		}
		if browserName != session.BrowserName {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "browser mismatch"})
			return
		}

		c.Set(consts.CTX_USER_ID, session.UserID)
		c.Set(consts.CTX_USERNAME, session.Username)
		c.Next()
	}
}
