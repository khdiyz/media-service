#!/bin/bash

# Proto generation script for media-service

set -e

# Colors for output
GREEN='\033[0;32m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

echo -e "${BLUE}Generating protobuf files...${NC}"

# Create genproto directory if it doesn't exist
mkdir -p genproto/media/v1

# Generate Go code from proto files
protoc \
  --go_out=. \
  --go_opt=paths=source_relative \
  --go-grpc_out=. \
  --go-grpc_opt=paths=source_relative \
  proto/media/v1/*.proto

echo -e "${GREEN}✓ Proto files generated successfully${NC}"
echo -e "${BLUE}Generated files:${NC}"
echo "  - genproto/media/v1/media.pb.go"
echo "  - genproto/media/v1/media_grpc.pb.go"
