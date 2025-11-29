package service

import (
	"bytes"
	"context"
	"io"

	"github.com/khdiyz/common/logger"
	"github.com/khdiyz/media-service/internal/storage"
)

// MediaService handles business logic for media operations
type MediaService struct {
	storage storage.Storage
	log     *logger.Logger
}

// NewMediaService creates a new MediaService
func NewMediaService(storage storage.Storage, log *logger.Logger) *MediaService {
	return &MediaService{
		storage: storage,
		log:     log,
	}
}

// UploadFile uploads a file to storage
func (s *MediaService) UploadFile(ctx context.Context, fileName string, content []byte, contentType string) (string, error) {
	reader := bytes.NewReader(content)
	fileSize := int64(len(content))

	return s.storage.Upload(ctx, fileName, fileSize, reader, contentType)
}

// UploadStream uploads a file from a reader (for streaming)
func (s *MediaService) UploadStream(ctx context.Context, fileName string, fileSize int64, reader io.Reader, contentType string) (string, error) {
	return s.storage.Upload(ctx, fileName, fileSize, reader, contentType)
}

// DownloadFile downloads a file from storage
func (s *MediaService) DownloadFile(ctx context.Context, filePath string) (io.ReadCloser, error) {
	return s.storage.Download(ctx, filePath)
}

// DeleteFile removes a file from storage
func (s *MediaService) DeleteFile(ctx context.Context, filePath string) error {
	return s.storage.Delete(ctx, filePath)
}

// GetFileURL returns the public URL for a file
func (s *MediaService) GetFileURL(filePath string) string {
	return s.storage.GetURL(filePath)
}
