package storage

import (
	"context"
	"io"
)

// Storage defines the interface for file storage operations
type Storage interface {
	// Upload uploads a file to storage and returns the file path/key
	Upload(ctx context.Context, fileName string, fileSize int64, reader io.Reader, contentType string) (string, error)

	// Download retrieves a file from storage
	Download(ctx context.Context, filePath string) (io.ReadCloser, error)

	// Delete removes a file from storage
	Delete(ctx context.Context, filePath string) error

	// GetURL returns the public URL for accessing a file
	GetURL(filePath string) string
}
