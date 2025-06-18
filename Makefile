.PHONY: build clean test docker-build docker-run help

# Default target
all: build

# Build the binary
build:
	go build -o yolink-exporter .

# Clean build artifacts
clean:
	rm -f yolink-exporter
	rm -f yolink-exporter-*

# Run tests
test:
	go test ./...

# Build Docker image
docker-build:
	docker build -t yolink-exporter .

# Run with Docker (requires .env file)
docker-run:
	docker-compose up -d

# Stop Docker containers
docker-stop:
	docker-compose down

# View Docker logs
docker-logs:
	docker-compose logs -f yolink-exporter

# Build for multiple platforms
build-all: clean
	GOOS=linux GOARCH=amd64 go build -o yolink-exporter-linux .
	GOOS=darwin GOARCH=amd64 go build -o yolink-exporter-darwin .
	GOOS=windows GOARCH=amd64 go build -o yolink-exporter-windows.exe .

# Install dependencies
deps:
	go mod download
	go mod tidy

# Format code
fmt:
	go fmt ./...

# Run linter
lint:
	golangci-lint run

# Show help
help:
	@echo "Available targets:"
	@echo "  build       - Build the binary"
	@echo "  clean       - Clean build artifacts"
	@echo "  test        - Run tests"
	@echo "  docker-build- Build Docker image"
	@echo "  docker-run  - Run with Docker Compose"
	@echo "  docker-stop - Stop Docker containers"
	@echo "  docker-logs - View Docker logs"
	@echo "  build-all   - Build for multiple platforms"
	@echo "  deps        - Install dependencies"
	@echo "  fmt         - Format code"
	@echo "  lint        - Run linter"
	@echo "  help        - Show this help" 