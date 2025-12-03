package cronjobiam

import (
	"time"

	"github.com/forbearing/gst/database"
	modeliam "github.com/forbearing/gst/internal/model/iam"
	"github.com/forbearing/gst/logger"
)

// CleanupOnlineUser cleanups the online user that not active for 1 minute.
func CleanupOnlineUser() error {
	end := time.Now().Add(-1 * time.Minute)
	ous := make([]*modeliam.OnlineUser, 0)

	if err := database.Database[*modeliam.OnlineUser](nil).WithTimeRange("updated_at", time.Time{}, end).List(&ous); err != nil {
		logger.Cronjob.Error(err)
	}
	if err := database.Database[*modeliam.OnlineUser](nil).WithPurge().Delete(ous...); err != nil {
		logger.Cronjob.Error(err)
	}
	return nil
}
