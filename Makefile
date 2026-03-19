.PHONY: build install uninstall clean test test-coverage test-coverage-html test-docker fmt vet lint deps package ci-build release-test install-hooks help

BINARY_NAME=yum-bundle
BUILD_DIR=build
GO=go
VERSION := $(shell cat VERSION | tr -d '[:space:]').0
COMMIT  := $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
GOFLAGS=-ldflags="-s -w \
  -X github.com/yum-bundle/yum-bundle/internal/commands.version=$(VERSION) \
  -X github.com/yum-bundle/yum-bundle/internal/commands.commit=$(COMMIT)"
INSTALL_DIR ?= /usr/local/bin
USE_SUDO ?= sudo

# Show help
help:
	@echo "Available targets:"
	@echo "  build               - Build the binary"
	@echo "  install             - Install the binary to $(INSTALL_DIR) (may require sudo)"
	@echo "  uninstall           - Remove the binary from $(INSTALL_DIR)"
	@echo "  clean               - Remove build artifacts and coverage reports"
	@echo "  test                - Run tests"
	@echo "  test-coverage       - Run tests with coverage report"
	@echo "  test-coverage-html  - Run tests with HTML coverage report"
	@echo "  fmt                 - Format code"
	@echo "  vet                 - Run go vet"
	@echo "  lint                - Run golangci-lint"
	@echo "  deps                - Download and tidy dependencies"
	@echo "  package             - Build .rpm packages locally using nfpm"
	@echo "  ci-build            - Build and package like CI release job (cross-arch)"
	@echo "  test-docker         - Run tests in Docker (replicates CI fedora-latest)"
	@echo "  release-test        - Test release workflow locally (dry-run)"
	@echo "  install-hooks       - Install git pre-commit hook (format, lint, build)"
	@echo "  help                - Show this help message"
	@echo ""
	@echo "Environment variables:"
	@echo "  INSTALL_DIR - Installation directory (default: /usr/local/bin)"
	@echo "  USE_SUDO    - Command prefix for install/uninstall (default: sudo)"

# Build the binary
build:
	@echo "Building $(BINARY_NAME)..."
	@mkdir -p $(BUILD_DIR)
	CGO_ENABLED=0 $(GO) build $(GOFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME) ./cmd/yum-bundle
	@echo "✓ Binary built at $(BUILD_DIR)/$(BINARY_NAME)"

# Install the binary to $(INSTALL_DIR)
install: build
	@echo "Installing $(BINARY_NAME) to $(INSTALL_DIR)..."
	@$(USE_SUDO) cp $(BUILD_DIR)/$(BINARY_NAME) $(INSTALL_DIR)/
	@$(USE_SUDO) chmod +x $(INSTALL_DIR)/$(BINARY_NAME)
	@echo "✓ $(BINARY_NAME) installed successfully to $(INSTALL_DIR)"

# Uninstall the binary
uninstall:
	@echo "Uninstalling $(BINARY_NAME) from $(INSTALL_DIR)..."
	@$(USE_SUDO) rm -f $(INSTALL_DIR)/$(BINARY_NAME)
	@echo "✓ $(BINARY_NAME) uninstalled from $(INSTALL_DIR)"

# Clean build artifacts
clean:
	@echo "Cleaning build artifacts..."
	@rm -rf $(BUILD_DIR)
	@rm -rf dist
	@rm -f coverage.out coverage.html
	@echo "✓ Clean complete"

# Run tests
test:
	@echo "Running tests..."
	$(GO) test -v -race -p 1 ./...

# Run tests with coverage
test-coverage:
	@echo "Running tests with coverage..."
	$(GO) test -v -race -coverprofile=coverage.out -covermode=atomic ./...
	@echo "Coverage report:"
	$(GO) tool cover -func=coverage.out

# Run tests with coverage and generate HTML report
test-coverage-html: test-coverage
	@echo "Generating HTML coverage report..."
	$(GO) tool cover -html=coverage.out -o coverage.html
	@echo "✓ Coverage report generated at coverage.html"

# Format code
fmt:
	@echo "Formatting code..."
	$(GO) fmt ./...

# Run go vet
vet:
	@echo "Running go vet..."
	$(GO) vet ./...

# Run golangci-lint
lint:
	@echo "Running linter..."
	$(GO) run github.com/golangci/golangci-lint/cmd/golangci-lint@latest run

# Download dependencies
deps:
	@echo "Downloading dependencies..."
	$(GO) mod download
	$(GO) mod tidy
	@$(GO) mod verify

# Build .rpm packages locally using nfpm
package: build
	@echo "Building .rpm packages..."
	@if ! command -v nfpm >/dev/null 2>&1; then \
		echo "Installing nfpm..."; \
		$(GO) install github.com/goreleaser/nfpm/v2/cmd/nfpm@latest; \
	fi
	@mkdir -p dist
	@VERSION=$$(cat VERSION | tr -d '[:space:]').0; \
	echo "Building packages for version $$VERSION"; \
	for arch in x86_64 aarch64 armv7hl; do \
		echo "Building for $$arch..."; \
		case $$arch in \
			x86_64)  GOARCH=amd64 GOARM= ;; \
			aarch64) GOARCH=arm64 GOARM= ;; \
			armv7hl) GOARCH=arm   GOARM=7 ;; \
		esac; \
		CGO_ENABLED=0 GOOS=linux GOARCH=$$GOARCH GOARM=$$GOARM \
			$(GO) build $(GOFLAGS) -o $(BUILD_DIR)/yum-bundle ./cmd/yum-bundle; \
		NFPM_VERSION=$$VERSION NFPM_ARCH=$$arch \
		nfpm package \
			--config .nfpm.yaml \
			--target dist/ \
			--packager rpm || true; \
	done
	@echo "✓ Packages built in dist/"

# Build and package like CI release job (cross-arch)
# Usage: make ci-build [ARCH=x86_64] [VERSION=0.1.0]
ci-build:
	@echo "Testing CI build step locally..."
	@if ! command -v nfpm >/dev/null 2>&1; then \
		echo "Installing nfpm..."; \
		$(GO) install github.com/goreleaser/nfpm/v2/cmd/nfpm@latest; \
	fi
	@ARCH=$${ARCH:-x86_64}; \
	VERSION=$${VERSION:-$$(cat VERSION | tr -d '[:space:]').0}; \
	case $$ARCH in \
		x86_64)  GOARCH=amd64 GOARM= RPMARCH=x86_64   ;; \
		aarch64) GOARCH=arm64 GOARM= RPMARCH=aarch64   ;; \
		armv7hl) GOARCH=arm   GOARM=7 RPMARCH=armv7hl  ;; \
		*) echo "Unknown architecture: $$ARCH"; exit 1 ;; \
	esac; \
	echo "Building for architecture: $$ARCH (GOARCH=$$GOARCH, RPMARCH=$$RPMARCH)"; \
	echo "Version: $$VERSION"; \
	mkdir -p build dist artifacts; \
	CGO_ENABLED=0 GOOS=linux GOARCH=$$GOARCH GOARM=$$GOARM \
		$(GO) build $(GOFLAGS) -o build/yum-bundle ./cmd/yum-bundle; \
	NFPM_VERSION=$$VERSION NFPM_ARCH=$$RPMARCH \
	nfpm package \
		--config .nfpm.yaml \
		--target dist/ \
		--packager rpm; \
	echo "✓ CI test complete."

# Run tests in Docker (Fedora-based, mirrors CI)
test-docker:
	@echo "Running tests in Docker (CI-like environment, Fedora + Go)..."
	@docker run --rm -v "$$(pwd)":/app -w /app fedora:39 bash -c '\
		dnf install -y golang gcc && \
		cd /app && go test -race -v -p 1 -failfast ./...'

# Test release workflow locally (dry-run)
release-test:
	@echo "Testing release workflow..."
	@echo "VERSION file contents: $$(cat VERSION)"
	@echo "This would calculate next patch version based on existing releases"
	@echo "Run 'make package' to build packages locally"

# Install git pre-commit hook
install-hooks:
	@echo "Installing git pre-commit hook..."
	@chmod +x .githooks/pre-commit
	@git config core.hooksPath .githooks
	@echo "✓ Pre-commit hook installed. Run on: git commit (when Go files are staged)"
