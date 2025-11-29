package handler

import (
	"context"
	"io"
	"time"

	"github.com/khdiyz/common/logger"
	"github.com/khdiyz/media-service/internal/service"
	mediav1 "github.com/khdiyz/media-service/proto/media/v1"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// MediaHandler implements the gRPC MediaServiceServer
type MediaHandler struct {
	mediav1.UnimplementedMediaServiceServer
	service *service.MediaService
	log     *logger.Logger
}

// NewMediaHandler creates a new MediaHandler
func NewMediaHandler(service *service.MediaService, log *logger.Logger) *MediaHandler {
	return &MediaHandler{
		service: service,
		log:     log,
	}
}

// Upload uploads a file to storage
func (h *MediaHandler) Upload(ctx context.Context, req *mediav1.UploadRequest) (*mediav1.UploadResponse, error) {
	h.log.Infow("Upload request received", "file_name", req.FileName, "content_type", req.ContentType)

	filePath, err := h.service.UploadFile(ctx, req.FileName, req.Content, req.ContentType)
	if err != nil {
		h.log.Errorw("Failed to upload file", "error", err)
		return nil, status.Errorf(codes.Internal, "failed to upload file: %v", err)
	}

	url := h.service.GetFileURL(filePath)
	fileSize := int64(len(req.Content))

	return &mediav1.UploadResponse{
		FilePath:   filePath,
		Url:        url,
		FileSize:   fileSize,
		UploadedAt: time.Now().Format(time.RFC3339),
	}, nil
}

// UploadStream uploads a file using streaming
func (h *MediaHandler) UploadStream(stream mediav1.MediaService_UploadStreamServer) error {
	h.log.Info("UploadStream request received")

	// Read first message to get metadata
	req, err := stream.Recv()
	if err != nil {
		return status.Errorf(codes.InvalidArgument, "failed to receive metadata: %v", err)
	}

	metadata := req.GetMetadata()
	if metadata == nil {
		return status.Error(codes.InvalidArgument, "first message must be metadata")
	}

	// Create a pipe to stream data from gRPC to storage
	reader, writer := io.Pipe()

	// Channel to capture upload result/error
	type uploadResult struct {
		filePath string
		err      error
	}
	resultChan := make(chan uploadResult, 1)

	// Start upload in a goroutine
	go func() {
		filePath, err := h.service.UploadStream(context.Background(), metadata.FileName, metadata.FileSize, reader, metadata.ContentType)
		resultChan <- uploadResult{filePath: filePath, err: err}
	}()

	// Read chunks from stream and write to pipe
	var totalBytes int64
	for {
		req, err := stream.Recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			writer.CloseWithError(err)
			return status.Errorf(codes.Internal, "failed to receive chunk: %v", err)
		}

		chunk := req.GetChunk()
		if chunk == nil {
			continue
		}

		n, err := writer.Write(chunk)
		if err != nil {
			writer.CloseWithError(err)
			return status.Errorf(codes.Internal, "failed to write chunk: %v", err)
		}
		totalBytes += int64(n)
	}
	writer.Close()

	// Wait for upload to complete
	result := <-resultChan
	if result.err != nil {
		return status.Errorf(codes.Internal, "failed to upload file: %v", result.err)
	}

	url := h.service.GetFileURL(result.filePath)

	return stream.SendAndClose(&mediav1.UploadResponse{
		FilePath:   result.filePath,
		Url:        url,
		FileSize:   totalBytes,
		UploadedAt: time.Now().Format(time.RFC3339),
	})
}

// Download downloads a file from storage
func (h *MediaHandler) Download(ctx context.Context, req *mediav1.DownloadRequest) (*mediav1.DownloadResponse, error) {
	h.log.Infow("Download request received", "file_path", req.FilePath)

	reader, err := h.service.DownloadFile(ctx, req.FilePath)
	if err != nil {
		h.log.Errorw("Failed to download file", "file_path", req.FilePath, "error", err)
		return nil, status.Errorf(codes.NotFound, "file not found: %v", err)
	}
	defer reader.Close()

	content, err := io.ReadAll(reader)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to read file content: %v", err)
	}

	// Note: In a real scenario, we might want to store/retrieve content type and original filename
	// For now, we'll return generic values or what we can derive
	return &mediav1.DownloadResponse{
		FileName:    req.FilePath, // We don't store original name separately in this simple impl
		Content:     content,
		ContentType: "application/octet-stream", // Default
		FileSize:    int64(len(content)),
	}, nil
}

// DownloadStream downloads a file using streaming
func (h *MediaHandler) DownloadStream(req *mediav1.DownloadRequest, stream mediav1.MediaService_DownloadStreamServer) error {
	h.log.Infow("DownloadStream request received", "file_path", req.FilePath)

	reader, err := h.service.DownloadFile(stream.Context(), req.FilePath)
	if err != nil {
		return status.Errorf(codes.NotFound, "file not found: %v", err)
	}
	defer reader.Close()

	// Send metadata first
	err = stream.Send(&mediav1.DownloadStreamResponse{
		Data: &mediav1.DownloadStreamResponse_Metadata{
			Metadata: &mediav1.FileMetadata{
				FileName: req.FilePath,
			},
		},
	})
	if err != nil {
		return status.Errorf(codes.Internal, "failed to send metadata: %v", err)
	}

	// Stream chunks
	buffer := make([]byte, 32*1024) // 32KB chunks
	for {
		n, err := reader.Read(buffer)
		if n > 0 {
			err := stream.Send(&mediav1.DownloadStreamResponse{
				Data: &mediav1.DownloadStreamResponse_Chunk{
					Chunk: buffer[:n],
				},
			})
			if err != nil {
				return status.Errorf(codes.Internal, "failed to send chunk: %v", err)
			}
		}
		if err == io.EOF {
			break
		}
		if err != nil {
			return status.Errorf(codes.Internal, "failed to read file: %v", err)
		}
	}

	return nil
}

// Delete removes a file from storage
func (h *MediaHandler) Delete(ctx context.Context, req *mediav1.DeleteRequest) (*mediav1.DeleteResponse, error) {
	h.log.Infow("Delete request received", "file_path", req.FilePath)

	err := h.service.DeleteFile(ctx, req.FilePath)
	if err != nil {
		h.log.Errorw("Failed to delete file", "file_path", req.FilePath, "error", err)
		return nil, status.Errorf(codes.Internal, "failed to delete file: %v", err)
	}

	return &mediav1.DeleteResponse{
		Success: true,
		Message: "File deleted successfully",
	}, nil
}

// GetURL returns the public URL for a file
func (h *MediaHandler) GetURL(ctx context.Context, req *mediav1.GetURLRequest) (*mediav1.GetURLResponse, error) {
	url := h.service.GetFileURL(req.FilePath)
	return &mediav1.GetURLResponse{
		Url: url,
	}, nil
}

// GetFileInfo retrieves metadata about a file
func (h *MediaHandler) GetFileInfo(ctx context.Context, req *mediav1.GetFileInfoRequest) (*mediav1.GetFileInfoResponse, error) {
	// Since our simple storage doesn't store separate metadata, we'll return basic info
	// In a real app, we might check DB or object storage metadata
	url := h.service.GetFileURL(req.FilePath)

	return &mediav1.GetFileInfoResponse{
		FilePath: req.FilePath,
		Url:      url,
	}, nil
}
