package serviceaichat

import (
	"github.com/cockroachdb/errors"
	"github.com/forbearing/gst/database"
	modelaichat "github.com/forbearing/gst/internal/model/aichat"
	"github.com/forbearing/gst/provider/minio"
	"github.com/forbearing/gst/types"
)

// Upload handles the file upload process for attachments
func Upload(ctx *types.ServiceContext) error {
	fileHeader, err := ctx.FormFile("file")
	if err != nil {
		return errors.Wrap(err, "failed to get file from form")
	}
	// Try to get message_id from form, though it might be empty
	messageID := ctx.PostForm("message_id")
	if len(messageID) == 0 {
		return errors.New("message_id is required")
	}

	src, err := fileHeader.Open()
	if err != nil {
		return errors.Wrap(err, "failed to open file")
	}
	defer src.Close()

	// Upload to MinIO
	info, err := minio.Put(ctx.Context(), fileHeader.Filename, src, &minio.PutOptions{
		Size:        fileHeader.Size,
		ContentType: fileHeader.Header.Get("Content-Type"),
	})
	if err != nil {
		return errors.Wrap(err, "failed to upload to storage")
	}

	// Determine file type
	fileType := modelaichat.AttachmentTypeFile
	contentType := fileHeader.Header.Get("Content-Type")
	// Simple detection logic, can be improved
	if len(contentType) >= 5 && contentType[:5] == "image" {
		fileType = modelaichat.AttachmentTypeImage
	} else if len(contentType) >= 5 && contentType[:5] == "video" {
		fileType = modelaichat.AttachmentTypeVideo
	} else if len(contentType) >= 5 && contentType[:5] == "audio" {
		fileType = modelaichat.AttachmentTypeAudio
	}

	// Create attachment record
	attachment := &modelaichat.Attachment{
		MessageID:   messageID,
		FileName:    fileHeader.Filename,
		FileSize:    fileHeader.Size,
		FileType:    fileType,
		MimeType:    contentType,
		StoragePath: info.Key,
		// DownloadURL can be generated dynamically
	}

	// Create record in database
	if err := database.Database[*modelaichat.Attachment](ctx.DatabaseContext()).Create(attachment); err != nil {
		// Attempt to cleanup storage if DB create fails
		_ = minio.Remove(ctx.Context(), info.Key)
		return errors.Wrap(err, "failed to create attachment record")
	}

	return nil
}
