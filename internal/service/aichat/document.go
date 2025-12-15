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

type Document struct {
	service.Base[*modelaichat.Document, *modelaichat.Document, *modelaichat.Document]
}

func (s *Document) CreateBefore(ctx *types.ServiceContext, doc *modelaichat.Document) error {
	return s.validateDocument(ctx.Context(), doc)
}

func (s *Document) CreateManyBefore(ctx *types.ServiceContext, docs ...*modelaichat.Document) error {
	for _, doc := range docs {
		if err := s.validateDocument(ctx.Context(), doc); err != nil {
			return err
		}
	}
	return nil
}

func (s *Document) validateDocument(ctx context.Context, doc *modelaichat.Document) error {
	if doc.FilePath == "" {
		// Some documents might be just text without a file?
		// If type is text, FilePath might be empty.
		if doc.Type != modelaichat.DocumentTypeText && doc.Type != "" {
			return errors.New("file_path is required for non-text documents")
		}
		return nil
	}

	// Verify if the file exists in MinIO
	exists, err := minio.Exists(ctx, doc.FilePath)
	if err != nil {
		return err
	}
	if !exists {
		return errors.Newf("file not found in storage: %s", doc.FilePath)
	}

	return nil
}

func (s *Document) DeleteAfter(ctx *types.ServiceContext, doc *modelaichat.Document) error {
	if doc.FilePath == "" {
		return nil
	}

	// Delete the file from MinIO
	err := minio.Remove(context.Background(), doc.FilePath)
	if err != nil {
		s.Logger.Errorz("failed to remove document file", zap.Error(err), zap.String("path", doc.FilePath))
		return nil
	}
	return nil
}

func (s *Document) DeleteManyAfter(ctx *types.ServiceContext, docs ...*modelaichat.Document) error {
	for _, doc := range docs {
		if err := s.DeleteAfter(ctx, doc); err != nil {
			s.Logger.Errorz("failed to delete document file", zap.Error(err), zap.String("id", doc.ID))
		}
	}
	return nil
}
