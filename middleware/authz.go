package middleware

import (
	"github.com/forbearing/gst/authz/rbac"
	"github.com/forbearing/gst/logger"
	. "github.com/forbearing/gst/response"
	"github.com/forbearing/gst/types/consts"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// Authz authorizes requests using RBAC.
// It derives subject from context or headers, falling back to system user.
func Authz() gin.HandlerFunc {
	return func(c *gin.Context) {
		var allow bool
		var err error
		sub := c.GetString(consts.CTX_USERNAME)
		obj := c.Request.URL.Path
		act := c.Request.Method

		// The "root" and "admin" is super admin user, can access all resources
		// If subject is not "root" or "admin", use user id as subject
		if sub != consts.AUTHZ_USER_ROOT && sub != consts.AUTHZ_USER_ADMIN {
			sub = c.GetString(consts.CTX_USER_ID)
		}
		if len(sub) == 0 {
			if h := c.GetHeader("X-Username"); len(h) > 0 {
				sub = h
			}
		}
		if len(sub) == 0 {
			if h := c.GetHeader("X-User-Id"); len(h) > 0 {
				sub = h
			}
		}
		if len(sub) == 0 {
			sub = consts.AUTHZ_USER_BLOCKED
		}
		if allow, err = rbac.Enforcer.Enforce(sub, obj, act); err != nil {
			zap.S().Error(err)
			ResponseJSON(c, CodeFailure)
			c.Abort()
			return
		}
		if allow {
			c.Next()
			logger.Authz.Infoz("",
				zap.String("sub", sub),
				zap.String("obj", obj),
				zap.String("act", act),
				zap.String("eft", string(consts.EffectAllow)),
			)
		} else {
			ResponseJSON(c, CodeForbidden)
			c.Abort()
			logger.Authz.Infoz("",
				zap.String("sub", sub),
				zap.String("obj", obj),
				zap.String("act", act),
				zap.String("eft", string(consts.EffectDeny)),
			)
			return

		}
		c.Next()
	}
}
