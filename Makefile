.PHONY: test lint install-lint install-mockery update-mocks fmt fmt-check build build-all build-linux build-darwin build-windows clean

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
GOLANGCI_LINT_VERSION=v1.61.0

# Mockery version
MOCKERY_VERSION=v3.6.1

# CGO flags for Bitwarden SDK
export CGO_ENABLED=1
export CGO_LDFLAGS=-lm

# Test with coverage
test:
	$(GOTEST) -v -race -coverprofile=coverage.out ./...
	$(GOCMD) tool cover -func=coverage.out

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

# Build for Linux (amd64) - native build on Linux
build-linux:
	mkdir -p $(BINARY_DIR)
	CGO_ENABLED=1 CGO_LDFLAGS="-lm" GOOS=linux GOARCH=amd64 $(GOBUILD) $(LDFLAGS) -o $(BINARY_DIR)/$(BINARY_NAME)-linux-amd64 $(CMD_PATH)
	@echo "Built $(BINARY_DIR)/$(BINARY_NAME)-linux-amd64"

# Build for macOS using Docker (requires osxcross SDK - complex setup)
# For now, macOS builds should be done on a Mac or via GitHub Actions
build-darwin-docker:
	@echo "⚠️  macOS cross-compilation with CGO requires osxcross SDK."
	@echo "    Build on macOS directly or use GitHub Actions."
	@echo ""
	@echo "    On macOS, run: make build-darwin"

# Native macOS build (run on macOS only)
build-darwin:
	mkdir -p $(BINARY_DIR)
	CGO_ENABLED=1 CGO_LDFLAGS="-lm" GOOS=darwin GOARCH=amd64 $(GOBUILD) $(LDFLAGS) -o $(BINARY_DIR)/$(BINARY_NAME)-darwin-amd64 $(CMD_PATH)
	CGO_ENABLED=1 CGO_LDFLAGS="-lm" GOOS=darwin GOARCH=arm64 $(GOBUILD) $(LDFLAGS) -o $(BINARY_DIR)/$(BINARY_NAME)-darwin-arm64 $(CMD_PATH)
	@echo "Built $(BINARY_DIR)/$(BINARY_NAME)-darwin-amd64"
	@echo "Built $(BINARY_DIR)/$(BINARY_NAME)-darwin-arm64"

# Build for Windows using Docker
build-windows-docker:
	mkdir -p $(BINARY_DIR)
	docker run --rm -v $(PWD):/app -w /app \
		--entrypoint "" \
		-e CGO_ENABLED=1 \
		-e GOOS=windows \
		-e GOARCH=amd64 \
		-e CC=x86_64-w64-mingw32-gcc \
		-e CXX=x86_64-w64-mingw32-g++ \
		goreleaser/goreleaser-cross:latest \
		go build -buildvcs=false $(LDFLAGS) -o $(BINARY_DIR)/$(BINARY_NAME)-windows-amd64.exe $(CMD_PATH)
	@echo "Built $(BINARY_DIR)/$(BINARY_NAME)-windows-amd64.exe"

# Build all platforms using Docker
build-all-docker: build-linux build-darwin-docker build-windows-docker
	@echo "All builds complete in $(BINARY_DIR)/"

# Build static binary (alias for build-linux)
build: build-linux

# Build for current platform (uses system CGO)
build-local:
	mkdir -p $(BINARY_DIR)
	CGO_ENABLED=1 CGO_LDFLAGS="-lm" $(GOBUILD) -o $(BINARY_DIR)/$(BINARY_NAME) $(CMD_PATH)
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
