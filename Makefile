.PHONY: test test-integration test-integration-build lint install-lint install-mockery update-mocks fmt fmt-check build build-all build-linux build-darwin build-windows clean help

# Go parameters
GOCMD=go
GOBUILD=$(GOCMD) build
GOCLEAN=$(GOCMD) clean
GOTEST=$(GOCMD) test
GOGET=$(GOCMD) get
GOMOD=$(GOCMD) mod
GOFMT=gofmt

# Build parameters
BINARY_NAME=vanish
BINARY_DIR=./bin
CMD_PATH=./cmd/vanish

# Build flags
LDFLAGS=-ldflags="-w -s"

# Linter version
GOLANGCI_LINT_VERSION=latest

# Mockery version
MOCKERY_VERSION=v3.6.1

# Test with coverage
test:
	$(GOTEST) -v -coverprofile=coverage.out -coverpkg=./internal/...,./pkg/...,./cmd/... ./...
	$(GOCMD) tool cover -func=coverage.out

# Test with race detector (no coverage)
test-race:
	$(GOTEST) -v -race ./...

# Integration tests using testcontainers-go
test-integration:
	@echo "Running integration tests with testcontainers-go..."
	@if [ -z "$$TS_TEST_AUTHKEY" ]; then \
		echo "Warning: TS_TEST_AUTHKEY not set. Some tests will be skipped."; \
	fi
	$(GOTEST) -v -tags=integration ./test/integration/... -timeout=15m

test-integration-build:
	@echo "Building integration test Docker images..."
	cd test/integration && docker build -t vanish-test-nas:latest -f Dockerfile.nas .

# Run linter
lint:
	golangci-lint run ./...

# Format all Go files
fmt:
	$(GOFMT) -s -w .

# Check if formatting is needed (fails if files need formatting)
fmt-check:
	@echo "Checking gofmt..."
	@test -z "$$($(GOFMT) -s -l .)" || (echo "Files need formatting:"; $(GOFMT) -s -l .; exit 1)
	@echo "All files are properly formatted"

# Install golangci-lint
install-lint:
	@if ! command -v golangci-lint &> /dev/null; then \
		curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $$(go env GOPATH)/bin $(GOLANGCI_LINT_VERSION); \
	else \
		echo "golangci-lint already installed"; \
	fi

# Install mockery v3.6.1
install-mockery:
	@if ! command -v mockery &> /dev/null; then \
		go install github.com/vektra/mockery/v3@$(MOCKERY_VERSION); \
	else \
		echo "mockery already installed"; \
	fi

# Generate mocks based on .mockery.yaml
update-mocks:
	mockery

# Build for all platforms (requires cross-compilers or Docker)
# For CGO cross-compilation, use: make build-all-docker
build-all: build-linux
	@echo ""
	@echo "Note: Cross-compilation for macOS/Windows requires Docker or cross-compilers."
	@echo "Run 'make build-all-docker' to build all platforms using Docker."
	@echo "Linux build complete in $(BINARY_DIR)/"

# Build for Linux (amd64) - pure Go, no CGO needed
build-linux:
	mkdir -p $(BINARY_DIR)
	GOOS=linux GOARCH=amd64 $(GOBUILD) $(LDFLAGS) -o $(BINARY_DIR)/$(BINARY_NAME)-linux-amd64 $(CMD_PATH)
	@echo "Built $(BINARY_DIR)/$(BINARY_NAME)-linux-amd64"

# Build for macOS (pure Go, no CGO needed - can cross-compile from Linux!)
build-darwin:
	mkdir -p $(BINARY_DIR)
	GOOS=darwin GOARCH=amd64 $(GOBUILD) $(LDFLAGS) -o $(BINARY_DIR)/$(BINARY_NAME)-darwin-amd64 $(CMD_PATH)
	GOOS=darwin GOARCH=arm64 $(GOBUILD) $(LDFLAGS) -o $(BINARY_DIR)/$(BINARY_NAME)-darwin-arm64 $(CMD_PATH)
	@echo "Built $(BINARY_DIR)/$(BINARY_NAME)-darwin-amd64"
	@echo "Built $(BINARY_DIR)/$(BINARY_NAME)-darwin-arm64"

# Legacy alias for compatibility
build-darwin-docker: build-darwin
	@echo "Note: Docker no longer required for macOS builds (pure Go now)"

# Build for Windows (pure Go, no CGO needed - no Docker required!)
build-windows:
	mkdir -p $(BINARY_DIR)
	GOOS=windows GOARCH=amd64 $(GOBUILD) $(LDFLAGS) -o $(BINARY_DIR)/$(BINARY_NAME)-windows-amd64.exe $(CMD_PATH)
	@echo "Built $(BINARY_DIR)/$(BINARY_NAME)-windows-amd64.exe"

# Build all platforms (pure Go - no Docker needed!)
build-all: build-linux build-darwin build-windows
	@echo "All builds complete in $(BINARY_DIR)/"

# Build static binary (alias for build-linux)
build: build-linux

# Build for current platform (pure Go)
build-local:
	mkdir -p $(BINARY_DIR)
	$(GOBUILD) $(LDFLAGS) -o $(BINARY_DIR)/$(BINARY_NAME) $(CMD_PATH)
	@echo "Binary built at $(BINARY_DIR)/$(BINARY_NAME)"

# Clean build artifacts
clean:
	$(GOCLEAN)
	rm -rf ./bin
	rm -f coverage.out

# Download dependencies
deps:
	$(GOMOD) download
	$(GOMOD) tidy

# Run the application
run:
	$(GOCMD) run $(CMD_PATH)/main.go

# Help
help:
	@echo "Available targets:"
	@echo "  test              - Run tests with coverage"
	@echo "  test-integration  - Run integration tests (requires Docker & TS_TEST_AUTHKEY)"
	@echo "  test-integration-build - Build Docker images for integration tests"
	@echo "  lint              - Run golangci-lint"
	@echo "  fmt               - Format code with gofmt"
	@echo "  fmt-check         - Check if code needs formatting (CI)"
	@echo "  install-lint      - Install golangci-lint"
	@echo "  install-mockery   - Install mockery"
	@echo "  update-mocks      - Generate mocks from .mockery.yaml"
	@echo ""
	@echo "Build targets:"
	@echo "  build-local       - Build for current platform"
	@echo "  build-linux       - Build for Linux amd64"
	@echo "  build-all-docker  - Build for all platforms using Docker"
	@echo "  build-darwin-docker - Build for macOS using Docker"
	@echo "  build-windows-docker- Build for Windows using Docker"
	@echo ""
	@echo "  clean             - Clean build artifacts"
	@echo "  deps              - Download and tidy dependencies"
	@echo "  run               - Run the application"
