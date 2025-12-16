package modelaichat

import "github.com/forbearing/gst/model"

// UserPreference represents user preferences and settings
type UserPreference struct {
	UserID string `gorm:"size:100;unique;not null;index" json:"user_id" schema:"user_id"` // User ID

	// UI preferences
	Theme    string `gorm:"size:20;default:light" json:"theme,omitempty" schema:"theme"`    // Theme (light/dark/auto)
	Language string `gorm:"size:10;default:en" json:"language,omitempty" schema:"language"` // Language (en/zh/ja)
	FontSize string `gorm:"size:20;default:medium" json:"font_size,omitempty"`              // Font size (small/medium/large)

	// Chat preferences
	DefaultModelID   *string `gorm:"size:100" json:"default_model_id,omitempty" schema:"default_model_id"` // Default model ID
	EnableAutoTitle  *bool   `gorm:"default:true" json:"enable_auto_title,omitempty"`                      // Whether to enable auto title generation
	EnableSoundNotif *bool   `gorm:"default:true" json:"enable_sound_notif,omitempty"`                     // Whether to enable sound notification
	StreamResponse   *bool   `gorm:"default:true" json:"stream_response,omitempty"`                        // Whether to stream response by default

	// Default model parameters
	DefaultTemperature *float64 `json:"default_temperature,omitempty"` // Default temperature (0-2)
	DefaultMaxTokens   *int     `json:"default_max_tokens,omitempty"`  // Default max tokens
	DefaultTopP        *float64 `json:"default_top_p,omitempty"`       // Default top_p

	model.Base
}

func (UserPreference) Purge() bool          { return true }
func (UserPreference) GetTableName() string { return "ai_user_preferences" }
