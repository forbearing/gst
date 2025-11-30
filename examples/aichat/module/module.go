// Package module provides business logic modules for the application.
//
// Recommended pattern:
//   - Organize each resource into its own subpackage under module/, e.g., module/user.
//   - Inside each subpackage, expose a Register() function that calls module.Use.
//   - Call these Register() functions from module.Init() to centralize startup.
//
// See module/helloworld for a complete example.
package module

import (
	"github.com/forbearing/gst/module/aichat"
)

func init() {
	aichat.Register()
}
