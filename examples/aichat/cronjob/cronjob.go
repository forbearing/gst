// Package cronjob provides scheduled task management for the application.
//
// Cron spec format: "second minute hour day month weekday" (6 fields)
// Examples: "0 0 2 * * *" (daily at 2:00 AM), "0 0 * * * *" (hourly)
//
// Example:
//
//	import "github.com/forbearing/gst/cronjob"
//
//	func cleanup() error {
//		// Implementation here
//		return nil
//	}
//
//	func init() {
//		cronjob.Register(cleanup, "0 0 2 * * *", "daily-cleanup")
//		// Optional: run immediately after registration
//		// cronjob.Register(cleanup, "0 0 2 * * *", "daily-cleanup", cronjob.Config{
//		//     RunImmediately: true,
//		// })
//	}
package cronjob

func init() {
	// TODO: Register your cron jobs here
	// Example:
	//   cronjob.Register(yourFunc, "0 0 * * * *", "hourly-task")
}
