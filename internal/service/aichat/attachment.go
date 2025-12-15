package serviceaichat

import (
	"context"

	"github.com/cockroachdb/errors"
	modelaichat "github.com/forbearing/gst/internal/model/aichat"
	"github.com/forbearing/gst/provider/minio"
	"github.com/forbearing/gst/service"
	"github.com/forbearing/gst/types"
	"go.uber.org/zap"
)

type Attachment struct {
	service.Base[*modelaichat.Attachment, *modelaichat.Attachment, *modelaichat.Attachment]
}

func (s *Attachment) CreateBefore(ctx *types.ServiceContext, att *modelaichat.Attachment) error {
	return s.validateAttachment(ctx.Context(), att)
}

func (s *Attachment) CreateManyBefore(ctx *types.ServiceContext, atts ...*modelaichat.Attachment) error {
	for _, att := range atts {
		if err := s.validateAttachment(ctx.Context(), att); err != nil {
			return err
		}
	}
	return nil
}

func (s *Attachment) validateAttachment(ctx context.Context, att *modelaichat.Attachment) error {
	if att.StoragePath == "" {
		return errors.New("storage_path is required")
	}

	// Verify if the file exists in MinIO
	exists, err := minio.Exists(ctx, att.StoragePath)
	if err != nil {
		return err
	}
	if !exists {
		return errors.Newf("file not found in storage: %s", att.StoragePath)
	}

	// If DownloadURL is empty, we could generate a presigned URL, but usually this is dynamic.
	// Or we can leave it empty and let the frontend request it via a separate endpoint or generate it here if it's a permanent link.
	// For now, we trust the input or leave it empty.

	return nil
}

func (s *Attachment) DeleteAfter(ctx *types.ServiceContext, att *modelaichat.Attachment) error {
	if att.StoragePath == "" {
		return nil
	}

	// Delete the file from MinIO
	// We use a background context or the request context.
	// Since this is DeleteAfter, the DB record is already deleted.
	// We should try to delete the file.
	err := minio.Remove(context.Background(), att.StoragePath)
	if err != nil {
		// Log error but don't fail the request as the DB record is gone.
		s.Logger.Errorz("failed to remove attachment file", zap.Error(err), zap.String("path", att.StoragePath))
		// We return nil to avoid confusing the user, as the primary resource (DB record) is deleted.
		return nil
	}
	return nil
}

func (s *Attachment) DeleteManyAfter(ctx *types.ServiceContext, atts ...*modelaichat.Attachment) error {
	for _, att := range atts {
		if err := s.DeleteAfter(ctx, att); err != nil {
			s.Logger.Errorz("failed to delete attachment file", zap.Error(err), zap.String("id", att.ID))
		}
	}
	return nil
}
