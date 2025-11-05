.PHONY: help build test fmt lint install clean release test-telemetry

APP_NAME=eigenx

VERSION_PKG=github.com/Layr-Labs/eigenx-cli/internal/version
TELEMETRY_PKG=github.com/Layr-Labs/eigenx-cli/pkg/telemetry
COMMON_PKG=github.com/Layr-Labs/eigenx-cli/pkg/common

# Get version from git tag, or use commit hash if no tags
VERSION ?= $(shell git describe --tags --exact-match 2>/dev/null || git describe --tags 2>/dev/null || git rev-parse --short HEAD 2>/dev/null || echo "unknown")
COMMIT_HASH := $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")

LD_FLAGS=\
  -X '$(VERSION_PKG).Version=$(VERSION)' \
  -X '$(VERSION_PKG).Commit=$(COMMIT_HASH)' \
  -X '$(TELEMETRY_PKG).embeddedTelemetryApiKey=$${TELEMETRY_TOKEN}' \
  -X '$(COMMON_PKG).embeddedEigenXReleaseVersion=$(VERSION)'

GO_PACKAGES=./pkg/... ./cmd/...
GO_TAGS ?=
ALL_FLAGS=
GO_FLAGS=-tags "$(GO_TAGS)" -ldflags "$(LD_FLAGS)"
GO=$(shell which go)
BIN=./bin
RELEASE_DIR=./release

help: ## Show available commands
	@echo "EigenX CLI - Available Commands:"
	@echo ""
	@grep -E '^[a-zA-Z0-9/_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-30s\033[0m %s\n", $$1, $$2}'

build: ## Build the binary
	@mkdir -p $(BIN)
	@echo "Building $(APP_NAME) version $(VERSION)..."
	@go build $(GO_FLAGS) -o $(BIN)/$(APP_NAME) cmd/$(APP_NAME)/main.go
	@echo "Build complete: $(BIN)/$(APP_NAME)"

test: ## Run tests
	@echo "Running tests..."
	$(GO) test -v ./... -p 1

test-fast: ## Run fast tests (skip slow integration tests)
	@echo "Running fast tests..."
	$(GO) test -v ./... -p 1 -timeout 5m -short

test-telemetry: ## Run telemetry tests
	@echo "Running telemetry tests..."
	$(GO) test -v $(TELEMETRY_PKG) -p 1

fmt: ## Format code
	@echo "Formatting code..."
	@go fmt $(GO_PACKAGES)

lint: ## Run linter
	@echo "Running linter..."
	@go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
	@golangci-lint run $(GO_PACKAGES)

install: build ## Install binary to ~/bin
	@mkdir -p ~/bin
	@cp $(BIN)/$(APP_NAME) ~/bin/
	@echo "Installed to ~/bin/$(APP_NAME)"

clean: ## Remove binaries and release artifacts
	@echo "Cleaning build artifacts..."
	@rm -rf $(BIN) $(RELEASE_DIR)
	@rm -f ~/bin/$(APP_NAME)

# Cross-compilation targets
build/darwin-arm64:
	@mkdir -p $(RELEASE_DIR)/darwin-arm64
	CGO_ENABLED=0 GOOS=darwin GOARCH=arm64 $(ALL_FLAGS) $(GO) build $(GO_FLAGS) -o $(RELEASE_DIR)/darwin-arm64/$(APP_NAME) cmd/$(APP_NAME)/main.go

build/darwin-amd64:
	@mkdir -p $(RELEASE_DIR)/darwin-amd64
	CGO_ENABLED=0 GOOS=darwin GOARCH=amd64 $(ALL_FLAGS) $(GO) build $(GO_FLAGS) -o $(RELEASE_DIR)/darwin-amd64/$(APP_NAME) cmd/$(APP_NAME)/main.go

build/linux-arm64:
	@mkdir -p $(RELEASE_DIR)/linux-arm64
	CGO_ENABLED=0 GOOS=linux GOARCH=arm64 $(ALL_FLAGS) $(GO) build $(GO_FLAGS) -o $(RELEASE_DIR)/linux-arm64/$(APP_NAME) cmd/$(APP_NAME)/main.go

build/linux-amd64:
	@mkdir -p $(RELEASE_DIR)/linux-amd64
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 $(ALL_FLAGS) $(GO) build $(GO_FLAGS) -o $(RELEASE_DIR)/linux-amd64/$(APP_NAME) cmd/$(APP_NAME)/main.go

build/windows-amd64:
	@mkdir -p $(RELEASE_DIR)/windows-amd64
	CGO_ENABLED=0 GOOS=windows GOARCH=amd64 $(ALL_FLAGS) $(GO) build $(GO_FLAGS) -o $(RELEASE_DIR)/windows-amd64/$(APP_NAME).exe cmd/$(APP_NAME)/main.go

build/windows-arm64:
	@mkdir -p $(RELEASE_DIR)/windows-arm64
	CGO_ENABLED=0 GOOS=windows GOARCH=arm64 $(ALL_FLAGS) $(GO) build $(GO_FLAGS) -o $(RELEASE_DIR)/windows-arm64/$(APP_NAME).exe cmd/$(APP_NAME)/main.go

release: clean ## Build release binaries for all platforms
	@echo "Building release binaries for version $(VERSION)..."
	@$(MAKE) build/darwin-arm64
	@$(MAKE) build/darwin-amd64
	@$(MAKE) build/linux-arm64
	@$(MAKE) build/linux-amd64
	@$(MAKE) build/windows-amd64
	@$(MAKE) build/windows-arm64
	@echo "Release builds complete in $(RELEASE_DIR)/"

release-check: release ## Build release and verify binaries
	@echo "Verifying release binaries..."
	@file $(RELEASE_DIR)/*/* 2>/dev/null || echo "Binaries created in $(RELEASE_DIR)"

# Development workflow
dev: fmt lint test build ## Development workflow: format, lint, test, build

pre-commit: fmt lint test-fast ## Pre-commit checks: format, lint, fast tests

.PHONY: build/darwin-arm64 build/darwin-amd64 build/linux-arm64 build/linux-amd64 build/windows-amd64 build/windows-arm64 release-check dev pre-commit
