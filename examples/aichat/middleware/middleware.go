// Package middleware provides custom HTTP middleware for the application.
//
// Register global middleware (applied to all routes) or authentication middleware
// (applied to authenticated routes only). Middlewares are automatically wrapped
// with tracing for performance monitoring.
//
// Example:
//
//	import (
//		"github.com/gin-gonic/gin"
//		"github.com/forbearing/gst/middleware"
//	)
//
//	func customMiddleware() gin.HandlerFunc {
//		return func(c *gin.Context) {
//			// Your middleware logic here
//			c.Next()
//		}
//	}
//
//	func init() {
//		// Register global middleware (applied to all routes)
//		middleware.Register(customMiddleware())
//
//		// Register authentication middleware (applied to authenticated routes only)
//		middleware.RegisterAuth(customMiddleware())
//	}
package middleware

func init() {
	// TODO: Register your custom middlewares here
	// Example:
	//   middleware.Register(yourMiddleware())
	//   middleware.RegisterAuth(yourAuthMiddleware())
}
