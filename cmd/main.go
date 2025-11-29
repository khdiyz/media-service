package main

import (
	"fmt"
	"net"
	"os"
	"os/signal"
	"syscall"

	"github.com/khdiyz/common/logger"
	"github.com/khdiyz/media-service/internal/config"
	"github.com/khdiyz/media-service/internal/handler"
	"github.com/khdiyz/media-service/internal/service"
	"github.com/khdiyz/media-service/internal/storage"
	mediav1 "github.com/khdiyz/media-service/proto/media/v1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

func main() {
	// Initialize logger
	log := logger.GetLogger()
	defer log.Sync()

	log.Info("Starting Media Service...")

	// Load configuration
	cfg := config.GetConfig(log)

	// Initialize Storage (MinIO)
	minioStorage, err := storage.NewMinioStorage(cfg, log)
	if err != nil {
		log.Fatalw("Failed to initialize MinIO storage", "error", err)
	}

	// Initialize Service
	mediaService := service.NewMediaService(minioStorage, log)

	// Initialize Handler
	mediaHandler := handler.NewMediaHandler(mediaService, log)

	// Start gRPC server
	listener, err := net.Listen("tcp", fmt.Sprintf("%s:%d", cfg.GrpcHost, cfg.GrpcPort))
	if err != nil {
		log.Fatalw("Failed to listen", "error", err)
	}

	grpcServer := grpc.NewServer()
	mediav1.RegisterMediaServiceServer(grpcServer, mediaHandler)

	// Enable reflection for debugging (e.g. using grpcurl)
	reflection.Register(grpcServer)

	// Graceful shutdown
	go func() {
		sigCh := make(chan os.Signal, 1)
		signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)
		<-sigCh
		log.Info("Shutting down gRPC server...")
		grpcServer.GracefulStop()
	}()

	log.Infow("Media Service started", "host", cfg.GrpcHost, "port", cfg.GrpcPort)
	if err := grpcServer.Serve(listener); err != nil {
		log.Fatalw("Failed to serve gRPC", "error", err)
	}
}
