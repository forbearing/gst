//nolint:predeclared
package new

import (
	"github.com/forbearing/gst/types/consts"
)

var modelContent = consts.CodeGeneratedComment() + `
package model

func init() {
}
`

var serviceContent = consts.CodeGeneratedComment() + `
package service

func init() {
}
`

var routerContent = consts.CodeGeneratedComment() + `
package router

func Init() error {
	return nil
}
`

var moduleContent = `// Package module provides business logic modules for the application.
//
// This package allows developers to implement and register custom business modules
// that handle specific domain logic and operations. You can either use modules
// provided by the gst framework or develop your own custom modules by implementing
// the Module interface, providing maximum flexibility and extensibility.
//
// Recommended pattern:
//   - Organize each resource into its own subpackage under module/, e.g., module/user.
//   - Inside each subpackage, expose a Register() function that wires models,
//     services, and routes via module.Use.
//   - Call these Register() functions from module.Init() to centralize startup.
//   - This mirrors the demo style (e.g., helloworld.Register()) and keeps
//     registration consistent and reusable.
//
// Example usage:
//
//	import (
//		"github.com/forbearing/gst/model"
//		"github.com/forbearing/gst/module"
//		"github.com/forbearing/gst/service"
//		"github.com/forbearing/gst/types"
//		"github.com/forbearing/gst/types/consts"
//	)
//
//	// Define your model
//	type User struct {
//		Name  string ` + "`json:\"name\"`" + `
//		Email string ` + "`json:\"email\"`" + `
//		model.Base
//	}
//
//	// Define custom request type (using Req suffix)
//	type UserReq struct {
//		Name  string ` + "`json:\"name\"`" + `
//		Email string ` + "`json:\"email\"`" + `
//	}
//
//	// Define custom response type (using Rsp suffix)
//	type UserRsp struct {
//		ID    string ` + "`json:\"id\"`" + `
//		Name  string ` + "`json:\"name\"`" + `
//		Email string ` + "`json:\"email\"`" + `
//	}
//
//	// Implement service
//	type UserService struct {
//		service.Base[*User, *UserReq, *UserRsp]
//	}
//
//	// Implement module (just implement the Module interface)
//	type UserModule struct{}
//
//	func (UserModule) Service() types.Service[*User, *UserReq, *UserRsp] {
//		return &UserService{}
//	}
//	func (UserModule) Pub() bool     { return false }
//	func (UserModule) Route() string { return "users" }
//	func (UserModule) Param() string { return "id" }
//
//	// In subpackage module/user:
//	// package user
//	// func Register() {
//	//     module.Use[*User, *UserReq, *UserRsp, *UserService](
//	//         &UserModule{},
//	//         consts.PHASE_CREATE,
//	//         consts.PHASE_DELETE,
//	//         consts.PHASE_UPDATE,
//	//         consts.PHASE_PATCH,
//	//         consts.PHASE_LIST,
//	//         consts.PHASE_GET,
//	//         consts.PHASE_CREATE_MANY,
//	//         consts.PHASE_DELETE_MANY,
//	//         consts.PHASE_UPDATE_MANY,
//	//         consts.PHASE_PATCH_MANY,
//	//     )
//	// }
//
//	// Then call user.Register() from module.Init().
//
// Place your business module implementations here.
package module

func init() {
    // TODO: Add your custom module registrations here
    //
    // Preferred approach:
    // - Create a subpackage per resource (e.g., module/user, module/order)
    // - In each subpackage, expose a Register() function that calls module.Use
    // - Call those Register() functions here to keep registrations centralized
    //
    // Example:
    //   // import "your/module/path/module/user"
    //   // user.Register()
    //
    // Direct registration is also supported if you prefer:
    // module.Use[*YourModel, *YourReq, *YourRsp, *YourService](
    //     &YourModule{},
    //     consts.PHASE_CREATE,
    //     consts.PHASE_DELETE,
    //     consts.PHASE_UPDATE,
    //     consts.PHASE_PATCH,
    //     consts.PHASE_LIST,
    //     consts.PHASE_GET,
    //     consts.PHASE_CREATE_MANY,
    //     consts.PHASE_DELETE_MANY,
    //     consts.PHASE_UPDATE_MANY,
    //     consts.PHASE_PATCH_MANY,
    // )
}
`

var mainContent = consts.CodeGeneratedComment() + `
package main

import (
	_ "%s/configx"
	_ "%s/cronjob"
	_ "%s/middleware"
	_ "%s/model"
	_ "%s/module"
	"%s/router"
	_ "%s/service"

	"github.com/forbearing/gst/bootstrap"
	. "github.com/forbearing/gst/util"
)

func main() {
	RunOrDie(bootstrap.Bootstrap)
	RunOrDie(router.Init)
	RunOrDie(bootstrap.Run)
}
`

const configxContent = `// Package configx provides custom configuration extensions for the application.
//
// This package is intended for developers to add their own configuration
// structures and register them with the configuration system using config.Register.
//
// Example usage:
//
//	import "github.com/forbearing/gst/config"
//
//	// Define custom configuration structs (without Config suffix)
//	type Payment struct {
//		Provider    string        ` + "`json:\"provider\" mapstructure:\"provider\" ini:\"provider\" yaml:\"provider\" default:\"alipay\"`" + `
//		MerchantID  string        ` + "`json:\"merchant_id\" mapstructure:\"merchant_id\" ini:\"merchant_id\" yaml:\"merchant_id\"`" + `
//		PrivateKey  string        ` + "`json:\"private_key\" mapstructure:\"private_key\" ini:\"private_key\" yaml:\"private_key\"`" + `
//		PublicKey   string        ` + "`json:\"public_key\" mapstructure:\"public_key\" ini:\"public_key\" yaml:\"public_key\"`" + `
//		NotifyURL   string        ` + "`json:\"notify_url\" mapstructure:\"notify_url\" ini:\"notify_url\" yaml:\"notify_url\"`" + `
//		ReturnURL   string        ` + "`json:\"return_url\" mapstructure:\"return_url\" ini:\"return_url\" yaml:\"return_url\"`" + `
//		Timeout     time.Duration ` + "`json:\"timeout\" mapstructure:\"timeout\" ini:\"timeout\" yaml:\"timeout\" default:\"30s\"`" + `
//		Enable      bool          ` + "`json:\"enable\" mapstructure:\"enable\" ini:\"enable\" yaml:\"enable\" default:\"false\"`" + `
//	}
//
//	type Email struct {
//		Host     string ` + "`json:\"host\" mapstructure:\"host\" ini:\"host\" yaml:\"host\" default:\"smtp.gmail.com\"`" + `
//		Port     int    ` + "`json:\"port\" mapstructure:\"port\" ini:\"port\" yaml:\"port\" default:\"587\"`" + `
//		Username string ` + "`json:\"username\" mapstructure:\"username\" ini:\"username\" yaml:\"username\"`" + `
//		Password string ` + "`json:\"password\" mapstructure:\"password\" ini:\"password\" yaml:\"password\"`" + `
//		From     string ` + "`json:\"from\" mapstructure:\"from\" ini:\"from\" yaml:\"from\"`" + `
//		Enable   bool   ` + "`json:\"enable\" mapstructure:\"enable\" ini:\"enable\" yaml:\"enable\" default:\"false\"`" + `
//	}
//
//	func init() {
//		// Register custom configurations
//		config.Register[Payment]()
//		config.Register[Email]()
//	}
//
//	// Usage in your application code:
//	func UsePaymentConfig() {
//		paymentCfg := config.Get[Payment]()
//		if paymentCfg.Enable {
//			// Use payment configuration
//			fmt.Printf("Payment Provider: %s\n", paymentCfg.Provider)
//		}
//	}
//
//	func UseEmailConfig() {
//		emailCfg := config.Get[Email]()
//		if emailCfg.Enable {
//			// Use email configuration
//			fmt.Printf("Email Host: %s:%d\n", emailCfg.Host, emailCfg.Port)
//		}
//	}
//
// Place your custom configuration code here.
package configx

func init() {
	// TODO: Register your custom configurations here
	// Examples:
	// config.Register[YourCustomConfig]()
	// config.Register[AnotherConfig]()
	//
	// Note: You need to import "github.com/forbearing/gst/config" when using config.Register
	//
	// Configuration values are loaded in priority order:
	// 1. Environment variables (format: SECTION_FIELD, e.g., PAYMENT_PROVIDER)
	// 2. Configuration file values
	// 3. Default values from struct tags
}
`

const cronjobContent = `// Package cronjob provides scheduled task management for the application.
//
// This package is intended for developers to register and manage cron jobs,
// scheduled tasks, and periodic operations using the built-in cronjob system.
//
// The cron specification format supports 6 fields: second minute hour day month weekday
// Examples:
//   - "0 30 * * * *"     - every 30 minutes
//   - "0 0 2 * * *"      - daily at 2:00 AM
//   - "0 0 0 * * 0"      - weekly on Sunday at midnight
//   - "*/10 * * * * *"   - every 10 seconds
//
// Example usage:
//
//	import "github.com/forbearing/gst/cronjob"
//
//	func init() {
//		// Register a daily cleanup task (at 2:00 AM every day)
//		cronjob.Register(cleanupTempFiles, "0 0 2 * * *", "daily-cleanup")
//
//		// Register an hourly health check (at the start of every hour)
//		cronjob.Register(healthCheck, "0 0 * * * *", "health-check")
//
//		// Register a task with custom config (run immediately + scheduled)
//		cronjob.Register(backupData, "0 0 0 * * 0", "weekly-backup", cronjob.Config{
//			RunImmediately: true,
//		})
//
//		// Register a task that runs every 30 seconds
//		cronjob.Register(monitorSystem, "*/30 * * * * *", "system-monitor")
//	}
//
//	func cleanupTempFiles() error {
//		// Implementation here
//		return nil
//	}
//
//	func healthCheck() error {
//		// Implementation here
//		return nil
//	}
//
//	func backupData() error {
//		// Implementation here
//		return nil
//	}
//
//	func monitorSystem() error {
//		// Implementation here
//		return nil
//	}
//
// Place your scheduled tasks and cron job registrations here.
package cronjob

func init() {
	// TODO: Add your cron job registrations here
	// Examples:
	// - Register periodic cleanup tasks
	// - Set up data synchronization jobs
	// - Schedule report generation
	// - Add health check routines
	//
	// Function signature: cronjob.Register(fn func() error, spec string, name string, config ...Config)
	// Cron spec format: "second minute hour day month weekday" (6 fields)
	// Config options: RunImmediately bool (whether to run immediately after registration)
}
`

const middlewareContent = `// Package middleware provides custom HTTP middleware for the application.
//
// This package is intended for developers to implement and register custom
// middleware functions that process HTTP requests and responses in the
// application pipeline.
//
// Middleware Registration:
//   - middleware.Register(...gin.HandlerFunc): Register global middleware (applied to all routes)
//   - middleware.RegisterAuth(...gin.HandlerFunc): Register authentication middleware (applied to authenticated routes only)
//
// Both functions automatically wrap registered middleware with tracing for performance monitoring.
//
// Example usage:
//
//	import (
//		"github.com/gin-gonic/gin"
//		"github.com/forbearing/gst/middleware"
//	)
//
//	func init() {
//		// Register global middleware (applied to all routes)
//		middleware.Register(
//			RequestLoggingMiddleware(),
//			RateLimitMiddleware(),
//			CORSMiddleware(),
//		)
//
//		// Register authentication middleware (applied to authenticated routes only)
//		middleware.RegisterAuth(
//			CustomAuthMiddleware(),
//			RoleBasedAccessMiddleware(),
//		)
//	}
//
//	// CustomAuthMiddleware validates API keys
//	func CustomAuthMiddleware() gin.HandlerFunc {
//		return func(c *gin.Context) {
//			apiKey := c.GetHeader("X-API-Key")
//			if !isValidAPIKey(apiKey) {
//				c.JSON(401, gin.H{"error": "Invalid API key"})
//				c.Abort()
//				return
//			}
//			c.Next()
//		}
//	}
//
//	// RequestLoggingMiddleware logs incoming requests
//	func RequestLoggingMiddleware() gin.HandlerFunc {
//		return func(c *gin.Context) {
//			// Log request details
//			c.Next()
//		}
//	}
//
//	// RateLimitMiddleware implements rate limiting
//	func RateLimitMiddleware() gin.HandlerFunc {
//		return func(c *gin.Context) {
//			// Implement rate limiting logic
//			c.Next()
//		}
//	}
//
//	// CORSMiddleware handles Cross-Origin Resource Sharing
//	func CORSMiddleware() gin.HandlerFunc {
//		return func(c *gin.Context) {
//			c.Header("Access-Control-Allow-Origin", "*")
//			c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
//			c.Header("Access-Control-Allow-Headers", "Content-Type, Authorization")
//
//			if c.Request.Method == "OPTIONS" {
//				c.AbortWithStatus(204)
//				return
//			}
//
//			c.Next()
//		}
//	}
//
//	// RoleBasedAccessMiddleware checks user roles
//	func RoleBasedAccessMiddleware() gin.HandlerFunc {
//		return func(c *gin.Context) {
//			// Check user roles and permissions
//			c.Next()
//		}
//	}
//
// Place your custom middleware implementations here.
package middleware

func init() {
	// Register global middleware (applied to all routes)
	// These middleware will be executed for every request
	// middleware.Register(
	//     RequestLoggingMiddleware(),
	//     RateLimitMiddleware(),
	//     CORSMiddleware(),
	// )

	// Register authentication middleware (applied to authenticated routes only)
	// These middleware will only be executed for routes that require authentication
	// middleware.RegisterAuth(
	//     CustomAuthMiddleware(),
	//     RoleBasedAccessMiddleware(),
	// )
	//
	// Note: You need to import the following packages when using middleware functions:
	// import (
	//     "github.com/gin-gonic/gin"
	//     "github.com/forbearing/gst/middleware"
	// )
}
`

const gitignoreContent = `# Binaries for programs and plugins
*.exe
*.exe~
*.dll
*.so
*.dylib

# Test binary, built with 'go test -c'
*.test

# Output of the go coverage tool, specifically when used with LiteIDE
*.out

# Dependency directories (remove the comment below to include it)
# vendor/

# Go workspace file
go.work

# IDE files
.vscode/
.idea/
*.swp
*.swo
*~

# OS generated files
.DS_Store
.DS_Store?
._*
.Spotlight-V100
.Trashes
ehthumbs.db
Thumbs.db

# Log files
*.log

# Temporary files
tmp/
temp/

# Build output
dist/
build/`
