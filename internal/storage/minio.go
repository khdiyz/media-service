package storage

import (
	"context"
	"fmt"
	"io"
	"path/filepath"
	"time"

	"github.com/google/uuid"
	"github.com/khdiyz/common/logger"
	"github.com/khdiyz/media-service/internal/config"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

// MinioStorage implements the Storage interface using MinIO
type MinioStorage struct {
	client     *minio.Client
	bucketName string
	fileURL    string
	log        *logger.Logger
}

// NewMinioStorage creates a new MinIO storage client
func NewMinioStorage(cfg *config.Config, log *logger.Logger) (Storage, error) {
	// Initialize MinIO client
	client, err := minio.New(cfg.MinioEndpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(cfg.MinioAccessKey, cfg.MinioSecretKey, ""),
		Secure: cfg.MinioUseSSL,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create minio client: %w", err)
	}

	storage := &MinioStorage{
		client:     client,
		bucketName: cfg.MinioBucketName,
		fileURL:    cfg.MinioFileUrl,
		log:        log,
	}

	// Ensure bucket exists
	if err := storage.ensureBucket(context.Background()); err != nil {
		return nil, fmt.Errorf("failed to ensure bucket exists: %w", err)
	}

	log.Infow("MinIO storage initialized successfully", "bucket", cfg.MinioBucketName)
	return storage, nil
}

// ensureBucket checks if bucket exists and creates it if it doesn't
func (m *MinioStorage) ensureBucket(ctx context.Context) error {
	exists, err := m.client.BucketExists(ctx, m.bucketName)
	if err != nil {
		return fmt.Errorf("failed to check bucket existence: %w", err)
	}

	if !exists {
		err = m.client.MakeBucket(ctx, m.bucketName, minio.MakeBucketOptions{})
		if err != nil {
			return fmt.Errorf("failed to create bucket: %w", err)
		}
		m.log.Infow("Created new bucket", "bucket", m.bucketName)
	}

	return nil
}

// Upload uploads a file to MinIO storage
func (m *MinioStorage) Upload(ctx context.Context, fileName string, fileSize int64, reader io.Reader, contentType string) (string, error) {
	// Generate unique file path with UUID to avoid collisions
	ext := filepath.Ext(fileName)
	uniqueID := uuid.NewString()
	timestamp := time.Now().Format("2006/01/02")
	filePath := fmt.Sprintf("%s/%s%s", timestamp, uniqueID, ext)

	// Upload file to MinIO
	_, err := m.client.PutObject(ctx, m.bucketName, filePath, reader, fileSize, minio.PutObjectOptions{
		ContentType: contentType,
	})
	if err != nil {
		m.log.Errorw("Failed to upload file to MinIO",
			"file_path", filePath,
			"error", err,
		)
		return "", fmt.Errorf("failed to upload file: %w", err)
	}

	m.log.Infow("File uploaded successfully",
		"file_path", filePath,
		"original_name", fileName,
		"size", fileSize,
	)

	return filePath, nil
}

// Download retrieves a file from MinIO storage
func (m *MinioStorage) Download(ctx context.Context, filePath string) (io.ReadCloser, error) {
	object, err := m.client.GetObject(ctx, m.bucketName, filePath, minio.GetObjectOptions{})
	if err != nil {
		m.log.Errorw("Failed to download file from MinIO",
			"file_path", filePath,
			"error", err,
		)
		return nil, fmt.Errorf("failed to download file: %w", err)
	}

	// Verify the object exists by checking stat
	_, err = object.Stat()
	if err != nil {
		object.Close()
		m.log.Errorw("File not found in MinIO",
			"file_path", filePath,
			"error", err,
		)
		return nil, fmt.Errorf("file not found: %w", err)
	}

	m.log.Infow("File downloaded successfully", "file_path", filePath)
	return object, nil
}

// Delete removes a file from MinIO storage
func (m *MinioStorage) Delete(ctx context.Context, filePath string) error {
	err := m.client.RemoveObject(ctx, m.bucketName, filePath, minio.RemoveObjectOptions{})
	if err != nil {
		m.log.Errorw("Failed to delete file from MinIO",
			"file_path", filePath,
			"error", err,
		)
		return fmt.Errorf("failed to delete file: %w", err)
	}

	m.log.Infow("File deleted successfully", "file_path", filePath)
	return nil
}

// GetURL returns the public URL for accessing a file
func (m *MinioStorage) GetURL(filePath string) string {
	if m.fileURL == "" {
		return ""
	}
	return fmt.Sprintf("%s/%s/%s", m.fileURL, m.bucketName, filePath)
}
