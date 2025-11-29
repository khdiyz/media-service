package config

import (
	"os"
	"sync"

	"github.com/joho/godotenv"
	"github.com/khdiyz/common/logger"
	"github.com/spf13/cast"
)

var (
	instance *Config
	once     sync.Once
)

type Config struct {
	GrpcHost string
	GrpcPort int

	MinioEndpoint   string
	MinioAccessKey  string
	MinioSecretKey  string
	MinioUseSSL     bool
	MinioBucketName string
	MinioFileUrl    string
}

func GetConfig(log *logger.Logger) *Config {
	once.Do(func() {
		if err := godotenv.Load(".env"); err != nil {
			log.Fatal(".env file not found")
		}

		instance = &Config{
			GrpcHost: cast.ToString(getOrReturnDefault("GRPC_HOST", "localhost")),
			GrpcPort: cast.ToInt(getOrReturnDefault("GRPC_PORT", 5051)),

			MinioEndpoint:   cast.ToString(getOrReturnDefault("MINIO_ENDPOINT", "")),
			MinioAccessKey:  cast.ToString(getOrReturnDefault("MINIO_ACCESS_KEY", "")),
			MinioSecretKey:  cast.ToString(getOrReturnDefault("MINIO_SECRET_KEY", "")),
			MinioUseSSL:     cast.ToBool(getOrReturnDefault("MINIO_USE_SSL", true)),
			MinioBucketName: cast.ToString(getOrReturnDefault("MINIO_BUCKET_NAME", "")),
			MinioFileUrl:    cast.ToString(getOrReturnDefault("MINIO_FILE_URL", "")),
		}
	})
	return instance
}

func getOrReturnDefault(key string, defaultValue any) any {
	val, exists := os.LookupEnv(key)
	if exists {
		return val
	}
	return defaultValue
}
