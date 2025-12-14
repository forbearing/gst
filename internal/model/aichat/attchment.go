package modelaichat

import "github.com/forbearing/gst/model"

// AttachmentType represents the type of attachment
type AttachmentType string

const (
	AttachmentTypeImage    AttachmentType = "image"    // Image attachment
	AttachmentTypeDocument AttachmentType = "document" // Document attachment (PDF, Word, TXT)
	AttachmentTypeAudio    AttachmentType = "audio"    // Audio attachment
	AttachmentTypeVideo    AttachmentType = "video"    // Video attachment
	AttachmentTypeFile     AttachmentType = "file"     // Generic file attachment
)

// Attachment represents an attachment associated with a message
type Attachment struct {
	MessageID    string         `gorm:"size:100;not null;index" json:"message_id" schema:"message_id"` // Message ID
	FileName     string         `gorm:"size:255;not null" json:"file_name" schema:"file_name"`         // File name
	FileSize     int64          `gorm:"not null" json:"file_size"`                                     // File size in bytes
	FileType     AttachmentType `gorm:"size:20;not null" json:"file_type" schema:"file_type"`          // File type
	MimeType     string         `gorm:"size:100" json:"mime_type,omitempty"`                           // MIME type
	StoragePath  string         `gorm:"size:500;not null" json:"storage_path,omitempty"`               // Storage path
	DownloadURL  string         `gorm:"size:500" json:"download_url,omitempty"`                        // Download URL
	ThumbnailURL *string        `gorm:"size:500" json:"thumbnail_url,omitempty"`                       // Thumbnail URL (for images/videos)
	Width        *int           `json:"width,omitempty"`                                               // Image width
	Height       *int           `json:"height,omitempty"`                                              // Image height
	Duration     *int           `json:"duration,omitempty"`                                            // Audio/video duration in seconds

	model.Base
}

func (Attachment) Purge() bool          { return true }
func (Attachment) GetTableName() string { return "ai_attachments" }
