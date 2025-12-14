package modelaichat

import "github.com/forbearing/gst/model"

// ImageGeneration represents an image generation record
type ImageGeneration struct {
	UserID        string `gorm:"size:100;not null;index" json:"user_id" schema:"user_id"` // User ID
	Prompt        string `gorm:"type:text;not null" json:"prompt"`                        // Generation prompt
	Model         string `gorm:"size:100" json:"model,omitempty" schema:"model"`          // Model name (e.g., dall-e-3, stable-diffusion)
	Size          string `gorm:"size:20" json:"size,omitempty"`                           // Image size (e.g., 1024x1024)
	Quality       string `gorm:"size:20" json:"quality,omitempty"`                        // Image quality (standard/hd)
	Style         string `gorm:"size:20" json:"style,omitempty"`                          // Image style (vivid/natural)
	ImageURL      string `gorm:"size:500" json:"image_url,omitempty"`                     // Generated image URL
	RevisedPrompt string `gorm:"type:text" json:"revised_prompt,omitempty"`               // Revised prompt (if applicable)

	model.Base
}

func (ImageGeneration) Purge() bool          { return true }
func (ImageGeneration) GetTableName() string { return "ai_image_generations" }
