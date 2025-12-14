package modelaichat

import (
	"time"

	"github.com/forbearing/gst/model"
)

// UserQuota represents user quota limits and usage
type UserQuota struct {
	UserID string `gorm:"size:100;unique;not null;index" json:"user_id" schema:"user_id"` // User ID

	// Quota limits
	DailyTokenLimit   *int64 `json:"daily_token_limit,omitempty"`   // Daily token limit
	MonthlyTokenLimit *int64 `json:"monthly_token_limit,omitempty"` // Monthly token limit
	DailyRequestLimit *int   `json:"daily_request_limit,omitempty"` // Daily request limit

	// Current usage
	TodayTokenUsed   int64 `gorm:"default:0" json:"today_token_used,omitempty"`   // Today's token usage
	MonthTokenUsed   int64 `gorm:"default:0" json:"month_token_used,omitempty"`   // Month's token usage
	TodayRequestUsed int   `gorm:"default:0" json:"today_request_used,omitempty"` // Today's request usage

	// Reset times
	LastDailyReset   *time.Time `json:"last_daily_reset,omitempty"`   // Last daily reset time
	LastMonthlyReset *time.Time `json:"last_monthly_reset,omitempty"` // Last monthly reset time

	model.Base
}

func (UserQuota) Purge() bool          { return true }
func (UserQuota) GetTableName() string { return "ai_user_quotas" }
