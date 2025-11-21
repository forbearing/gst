package middleware

import (
	"github.com/forbearing/gst/authz/rbac"
	"github.com/forbearing/gst/logger"
	. "github.com/forbearing/gst/response"
	"github.com/forbearing/gst/types/consts"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// Authz requires JwtAuth and must run after it, otherwise username will be empty
// and authorization will always be denied.
func Authz() gin.HandlerFunc {
	return func(c *gin.Context) {
		var allow bool
		var err error
		sub := c.GetString(consts.CTX_USERNAME)
		obj := c.Request.URL.Path
		act := c.Request.Method

		if sub != consts.ROOT && sub != consts.ADMIN {
			sub = c.GetString(consts.CTX_USER_ID)
		}
		if allow, err = rbac.Enforcer.Enforce(sub, obj, act); err != nil {
			zap.S().Error(err)
			ResponseJSON(c, CodeFailure)
			c.Abort()
			return
		}
		logger.Authz.Infoz("",
			zap.String("sub", sub),
			zap.String("obj", obj),
			zap.String("act", act),
			zap.Bool("res", allow),
		)
		if allow {
			c.Next()
		} else {
			ResponseJSON(c, CodeForbidden)
			c.Abort()
			return
		}
		c.Next()
	}
}
