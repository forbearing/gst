package logmgmt

import (
	"time"

	"github.com/forbearing/gst/database"
	"github.com/forbearing/gst/logger"
)

// cleanupLogs will delete logs older than 3 months
func cleanupLogs() error {
	end := time.Now().Add(-3 * 30 * 24 * time.Hour)
	oplogs := make([]*OperationLog, 0)

	if err := database.Database[*OperationLog](nil).WithTimeRange("created_at", time.Time{}, end).List(&oplogs); err != nil {
		logger.Cronjob.Error(err)
	}
	if err := database.Database[*OperationLog](nil).WithPurge().Delete(oplogs...); err != nil {
		logger.Cronjob.Error(err)
	}

	loginLogs := make([]*LoginLog, 0)
	if err := database.Database[*LoginLog](nil).WithTimeRange("created_at", time.Time{}, end).List(&loginLogs); err != nil {
		logger.Cronjob.Error(err)
	}
	if err := database.Database[*LoginLog](nil).WithPurge().Delete(loginLogs...); err != nil {
		logger.Cronjob.Error(err)
	}

	return nil
}
