// Package module provides business logic modules for the application.
//
// Recommended pattern:
//   - Organize each resource into its own subpackage under module/, e.g., module/user.
//   - Inside each subpackage, expose a Register() function that calls module.Use.
//   - Call these Register() functions from module.Init() to centralize startup.
//
// See module/helloworld for a complete example.
package module

func init() {
	// TODO: Call your module Register() functions here
	// Example:
	//   user.Register()
	//   order.Register()
}
