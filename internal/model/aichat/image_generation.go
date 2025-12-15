package modelaichat

import (
	"github.com/forbearing/gst/model"
)

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

// ImageGenerationReq represents a request for image generation
type ImageGenerationReq struct {
	Prompt         string `json:"prompt" binding:"required"`
	Model          string `json:"model,omitempty"`           // e.g. dall-e-3
	N              int    `json:"n,omitempty"`               // Number of images to generate
	Quality        string `json:"quality,omitempty"`         // standard or hd
	ResponseFormat string `json:"response_format,omitempty"` // url or b64_json
	Size           string `json:"size,omitempty"`            // e.g. 1024x1024
	Style          string `json:"style,omitempty"`           // vivid or natural
	User           string `json:"user,omitempty"`
}

// ImageGenerationRsp represents a response for image generation
type ImageGenerationRsp struct {
	Created int64                 `json:"created"`
	Data    []ImageGenerationData `json:"data"`
}

type ImageGenerationData struct {
	URL           string `json:"url,omitempty"`
	B64JSON       string `json:"b64_json,omitempty"`
	RevisedPrompt string `json:"revised_prompt,omitempty"`
}
