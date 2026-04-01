package middleware

import (
	"net/http"

	modeliamsession "github.com/forbearing/gst/internal/model/iam/session"
	"github.com/forbearing/gst/provider/redis"
	"github.com/forbearing/gst/types/consts"
	"github.com/gin-gonic/gin"
	"github.com/mssola/useragent"
)

// sessionRequiresPasswordChange reads the flag from session user info (set at login / updated on password change).
func sessionRequiresPasswordChange(session modeliamsession.Session) bool {
	if session.UserInfo == nil {
		return false
	}
	v, ok := session.UserInfo["must_change_password"].(bool)
	return ok && v
}

// mustChangePasswordExemptRoutes are allowed while MustChangePassword is true on the session.
func mustChangePasswordExempt(method, path string) bool {
	switch {
	case method == http.MethodPost && path == "/api/iam/change-password":
		return true
	case method == http.MethodPost && path == "/api/logout":
		return true
	case method == http.MethodGet && path == "/api/me":
		return true
	case method == http.MethodPost && path == "/api/heartbeat":
		return true
	default:
		return false
	}
}

func IAMSession() gin.HandlerFunc {
	return func(c *gin.Context) {
		// fmt.Println("----- identifySession middleware", c.Request.RequestURI)
		sessionID, err := c.Cookie("session_id")
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "no session"})
			return
		}
		// fmt.Println("----- SessionRedisKey", helper.SessionRedisKey(identity.SessionNamespace, sessionID))
		session, e := redis.Cache[modeliamsession.Session]().WithContext(c.Request.Context()).Get(modeliamsession.SessionRedisKey(modeliamsession.SessionNamespace, sessionID))
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

		if sessionRequiresPasswordChange(session) && !mustChangePasswordExempt(c.Request.Method, c.Request.URL.Path) {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{
				"error": "password change required before using this resource",
			})
			return
		}

		c.Set(consts.CTX_USER_ID, session.UserID)
		c.Set(consts.CTX_USERNAME, session.Username)
		c.Next()
	}
}
