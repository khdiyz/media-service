.PHONY: run proto-gen tidy build deps

# Default target
all: build

# Run the application
run:
	go run cmd/main.go

# Generate protobuf files
proto-gen:
	@./scripts/gen-proto.sh

# Tidy up dependencies
tidy:
	go mod tidy

# Build the application
build:
	go build ./...

# Install dependencies (optional helper)
deps:
	go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
	go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest
