// Package configx provides custom configuration extensions for the application.
//
// Define your custom configuration structs and register them using config.Register.
// See config.Register documentation for details on configuration loading priority
// and struct tag usage.
//
// Example:
//
//	import "github.com/forbearing/gst/config"
//
//	type Payment struct {
//		Provider string `json:"provider" mapstructure:"provider" default:"alipay"`
//		Enable   bool   `json:"enable" mapstructure:"enable" default:"false"`
//	}
//
//	func init() {
//		config.Register[Payment]()
//	}
package configx

func init() {
	// TODO: Register your custom configurations here
	// Example:
	//   config.Register[YourCustomConfig]()
}
