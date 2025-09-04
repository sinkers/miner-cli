# Makefile for miner-cli

# Variables
BINARY_NAME=miner-cli
BINARY_PATH=./$(BINARY_NAME)
MAIN_PACKAGE=.
GO_FILES=$(shell find . -name "*.go" -type f ! -name "*_test.go")
TEST_FILES=$(shell find . -name "*_test.go" -type f)

# Default Go settings
GOOS=$(shell go env GOOS)
GOARCH=$(shell go env GOARCH)

# Build flags
BUILD_FLAGS=-ldflags="-s -w" -trimpath
TEST_FLAGS=-v -race -coverprofile=coverage.out

# Colors for output
RED=\033[0;31m
GREEN=\033[0;32m
YELLOW=\033[1;33m
BLUE=\033[0;34m
NC=\033[0m # No Color

.PHONY: help build test test-verbose test-coverage test-benchmark clean deps tidy lint fmt vet install uninstall run-example docker-build docker-run cross-compile-all release

# Default target
all: deps fmt vet test build

help: ## Show this help message
	@echo "$(BLUE)Miner CLI - Available Commands:$(NC)"
	@echo ""
	@awk 'BEGIN {FS = ":.*##"; printf "\033[36m%-20s\033[0m %s\n", "Command", "Description"} /^[a-zA-Z_-]+:.*?##/ { printf "\033[36m%-20s\033[0m %s\n", $$1, $$2 }' $(MAKEFILE_LIST)

build: ## Build the binary
	@echo "$(BLUE)Building $(BINARY_NAME)...$(NC)"
	@go build $(BUILD_FLAGS) -o $(BINARY_PATH) $(MAIN_PACKAGE)
	@echo "$(GREEN)✓ Built $(BINARY_NAME) successfully$(NC)"

test: ## Run tests
	@echo "$(BLUE)Running tests...$(NC)"
	@go test ./... -v
	@echo "$(GREEN)✓ All tests passed$(NC)"

test-verbose: ## Run tests with verbose output
	@echo "$(BLUE)Running tests with verbose output...$(NC)"
	@go test ./... $(TEST_FLAGS)

test-coverage: ## Run tests with coverage
	@echo "$(BLUE)Running tests with coverage...$(NC)"
	@go test ./... $(TEST_FLAGS)
	@go tool cover -html=coverage.out -o coverage.html
	@echo "$(GREEN)✓ Coverage report generated: coverage.html$(NC)"

test-benchmark: ## Run benchmark tests
	@echo "$(BLUE)Running benchmark tests...$(NC)"
	@go test ./... -bench=. -benchmem

test-short: ## Run tests with short flag (skip long running tests)
	@echo "$(BLUE)Running short tests...$(NC)"
	@go test ./... -short

clean: ## Clean build artifacts and cache
	@echo "$(BLUE)Cleaning...$(NC)"
	@go clean
	@rm -f $(BINARY_NAME)
	@rm -f coverage.out coverage.html
	@rm -rf dist/
	@echo "$(GREEN)✓ Cleaned build artifacts$(NC)"

deps: ## Download dependencies
	@echo "$(BLUE)Downloading dependencies...$(NC)"
	@go mod download
	@echo "$(GREEN)✓ Dependencies downloaded$(NC)"

tidy: ## Clean up dependencies
	@echo "$(BLUE)Tidying dependencies...$(NC)"
	@go mod tidy
	@echo "$(GREEN)✓ Dependencies tidied$(NC)"

lint: ## Run linter (requires golangci-lint)
	@echo "$(BLUE)Running linter...$(NC)"
	@if command -v golangci-lint >/dev/null 2>&1; then \
		golangci-lint run; \
		echo "$(GREEN)✓ Linting completed$(NC)"; \
	else \
		echo "$(YELLOW)⚠ golangci-lint not installed. Install with: go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest$(NC)"; \
	fi

fmt: ## Format Go code
	@echo "$(BLUE)Formatting code...$(NC)"
	@go fmt ./...
	@echo "$(GREEN)✓ Code formatted$(NC)"

vet: ## Run go vet
	@echo "$(BLUE)Running go vet...$(NC)"
	@go vet ./...
	@echo "$(GREEN)✓ Vet check passed$(NC)"

install: build ## Install binary to GOPATH/bin
	@echo "$(BLUE)Installing $(BINARY_NAME)...$(NC)"
	@go install $(BUILD_FLAGS) $(MAIN_PACKAGE)
	@echo "$(GREEN)✓ $(BINARY_NAME) installed to $(shell go env GOPATH)/bin$(NC)"

uninstall: ## Remove installed binary
	@echo "$(BLUE)Uninstalling $(BINARY_NAME)...$(NC)"
	@rm -f $(shell go env GOPATH)/bin/$(BINARY_NAME)
	@echo "$(GREEN)✓ $(BINARY_NAME) uninstalled$(NC)"

run-example: build ## Build and run example command
	@echo "$(BLUE)Running example...$(NC)"
	@echo "$(YELLOW)Available commands: summary, devs, pools, stats, version, scan$(NC)"
	@echo "$(YELLOW)Example: ./$(BINARY_NAME) scan -i 127.0.0.1$(NC)"

# Development helpers
dev-setup: ## Setup development environment
	@echo "$(BLUE)Setting up development environment...$(NC)"
	@go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
	@go install golang.org/x/tools/cmd/goimports@latest
	@echo "$(GREEN)✓ Development tools installed$(NC)"

watch: ## Watch for file changes and run tests (requires entr)
	@echo "$(BLUE)Watching for changes...$(NC)"
	@if command -v entr >/dev/null 2>&1; then \
		find . -name "*.go" | entr -c make test; \
	else \
		echo "$(RED)✗ entr not installed. Install with: brew install entr (macOS) or apt-get install entr (Linux)$(NC)"; \
	fi

# Cross compilation
cross-compile: ## Cross compile for common platforms
	@echo "$(BLUE)Cross compiling...$(NC)"
	@mkdir -p dist
	@GOOS=linux GOARCH=amd64 go build $(BUILD_FLAGS) -o dist/$(BINARY_NAME)-linux-amd64 $(MAIN_PACKAGE)
	@GOOS=darwin GOARCH=amd64 go build $(BUILD_FLAGS) -o dist/$(BINARY_NAME)-darwin-amd64 $(MAIN_PACKAGE)
	@GOOS=darwin GOARCH=arm64 go build $(BUILD_FLAGS) -o dist/$(BINARY_NAME)-darwin-arm64 $(MAIN_PACKAGE)
	@GOOS=windows GOARCH=amd64 go build $(BUILD_FLAGS) -o dist/$(BINARY_NAME)-windows-amd64.exe $(MAIN_PACKAGE)
	@echo "$(GREEN)✓ Cross compilation completed. Binaries in dist/$(NC)"

cross-compile-all: ## Cross compile for all supported platforms
	@echo "$(BLUE)Cross compiling for all platforms...$(NC)"
	@mkdir -p dist
	@for os in darwin linux windows; do \
		for arch in amd64 arm64; do \
			if [ "$$os" = "windows" ] && [ "$$arch" = "arm64" ]; then continue; fi; \
			ext=""; \
			if [ "$$os" = "windows" ]; then ext=".exe"; fi; \
			echo "Building $$os/$$arch..."; \
			GOOS=$$os GOARCH=$$arch go build $(BUILD_FLAGS) -o dist/$(BINARY_NAME)-$$os-$$arch$$ext $(MAIN_PACKAGE); \
		done; \
	done
	@echo "$(GREEN)✓ Cross compilation completed for all platforms$(NC)"

# Docker targets
docker-build: ## Build Docker image
	@echo "$(BLUE)Building Docker image...$(NC)"
	@docker build -t $(BINARY_NAME):latest .
	@echo "$(GREEN)✓ Docker image built$(NC)"

docker-run: ## Run in Docker container
	@echo "$(BLUE)Running in Docker...$(NC)"
	@docker run --rm $(BINARY_NAME):latest --help

# Release preparation
release: clean test cross-compile-all ## Prepare release (clean, test, cross-compile)
	@echo "$(GREEN)✓ Release preparation completed$(NC)"
	@echo "$(BLUE)Release artifacts in dist/:$(NC)"
	@ls -la dist/

# Performance and profiling
profile-cpu: ## Run CPU profiling
	@echo "$(BLUE)Running CPU profile...$(NC)"
	@go test -cpuprofile=cpu.prof -bench=. ./...
	@echo "$(YELLOW)View with: go tool pprof cpu.prof$(NC)"

profile-mem: ## Run memory profiling
	@echo "$(BLUE)Running memory profile...$(NC)"
	@go test -memprofile=mem.prof -bench=. ./...
	@echo "$(YELLOW)View with: go tool pprof mem.prof$(NC)"

# Code quality
check: fmt vet lint test ## Run all quality checks
	@echo "$(GREEN)✓ All quality checks passed$(NC)"

pre-commit: fmt vet test ## Run pre-commit checks
	@echo "$(GREEN)✓ Pre-commit checks completed$(NC)"

# Information
info: ## Show project information
	@echo "$(BLUE)Project Information:$(NC)"
	@echo "Binary name: $(BINARY_NAME)"
	@echo "Go version: $(shell go version)"
	@echo "OS/Arch: $(GOOS)/$(GOARCH)"
	@echo "Go files: $(shell echo $(GO_FILES) | wc -w)"
	@echo "Test files: $(shell echo $(TEST_FILES) | wc -w)"
	@echo "Dependencies: $(shell go list -m all | wc -l)"

# Quick development cycle
quick: fmt test build ## Quick development cycle (format, test, build)
	@echo "$(GREEN)✓ Quick development cycle completed$(NC)"