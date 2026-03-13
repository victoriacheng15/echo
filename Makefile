BINARY_NAME=echo
BUILD_DIR=bin
LINT_IMAGE=ghcr.io/igorshubovych/markdownlint-cli:v0.44.0
GO_TAGS=-tags "sqlite_fts5"

# Container Engine (Default to Podman)
DOCKER ?= podman

# Dynamic Nix Detection
# 1. Check if nix-shell is available
# 2. Check if we are already in a nix-shell
# 3. Check if we are NOT in GitHub Actions
USE_NIX = $(shell if command -v nix-shell >/dev/null 2>&1 && [ -z "$$IN_NIX_SHELL" ] && [ "$$GITHUB_ACTIONS" != "true" ]; then echo "yes"; else echo "no"; fi)

# Determine the target(s) for the recursive call
TARGET_GOALS = $(if $(MAKECMDGOALS),$(MAKECMDGOALS),all)

ifeq ($(USE_NIX),yes)
    NIX_RUN = nix-shell --run
    # NIX_WRAP: Re-run the entire make command inside nix-shell and then exit the outer shell
    NIX_WRAP = @$(NIX_RUN) "make $(TARGET_GOALS)" && exit $$? ;
else
    NIX_RUN = bash -c
    # In the inner shell, NIX_WRAP is just the make silence prefix
    NIX_WRAP = @
endif

# Installation settings
PREFIX ?= $(shell echo $$HOME)/.local
BIN_DIR = $(PREFIX)/bin

.PHONY: all help update vet format test test-cov bench build build-web setup-tailwind clean install uninstall lint

# Default target: Run the full development lifecycle
all: update format vet test build

# Show help message
help:
	@echo "Usage: make [target]"
	@echo ""
	@echo "Available targets:"
	@echo "  all              - Run update, format, vet, test, and build"
	@echo "  help             - Show this help message"
	@echo "  update           - Run go mod tidy"
	@echo "  vet              - Run go vet"
	@echo "  format           - Run go fmt"
	@echo "  lint             - Run markdownlint via Docker"
	@echo "  test             - Run tests"
	@echo "  test-cov         - Run tests with coverage and open HTML report"
	@echo "  bench            - Run benchmarks"
	@echo "  build            - Build the binary under bin/"
	@echo "  build-web        - Build the static web application site into dist/"
	@echo "  setup-tailwind   - Download the tailwind css cli"
	@echo "  install          - Install the binary to $(BIN_DIR)"
	@echo "  uninstall        - Remove the binary from $(BIN_DIR)"
	@echo "  clean            - Remove build artifacts"

# Run markdownlint via Docker
lint:
	$(DOCKER) run --rm -v "$(PWD):/data:Z" -w /data $(LINT_IMAGE) --fix "**/*.md"

# Install the binary to the system
install: build
	@echo "Updating $(BINARY_NAME)..."
	mkdir -p $(BIN_DIR)
	install -m 755 $(BUILD_DIR)/$(BINARY_NAME) $(BIN_DIR)/$(BINARY_NAME)
	rm $(BUILD_DIR)/$(BINARY_NAME)
	@echo "Echo updated in $(BIN_DIR)"

# Remove the binary from the system
uninstall:
	rm -f $(BIN_DIR)/$(BINARY_NAME)
	@echo "Echo removed from $(BIN_DIR)"

# Run go mod tidy to update dependencies
update:
	echo "Updating dependencies..." && \
	go mod tidy

# Run go vet on all packages
vet:
	echo "Running go vet..." && \
	go vet $(GO_TAGS) ./...

# Run go fmt on all packages
format:
	echo "Running go fmt..." && \
	go fmt ./...

# Run tests for all packages
test:
	echo "Running tests..." && \
	go test $(GO_TAGS) ./...

# Run tests with coverage
test-cov:
	echo "Running tests with coverage..." && \
	go test $(GO_TAGS) -coverprofile=coverage.out ./... && \
	go tool cover -func=coverage.out && \
	rm -f coverage.out

# Run benchmarks
bench:
	echo "Running benchmarks..." && \
	go test $(GO_TAGS) -bench=. -benchmem ./...

# Build the binary under bin/
build:
	echo "Building binary..." && \
	mkdir -p $(BUILD_DIR) && \
	go build $(GO_TAGS) -o $(BUILD_DIR)/$(BINARY_NAME) ./cmd/mcp

# Build the static web application site into dist/
web-build: setup-tailwind
	echo "Building static web application..." && \
	rm -rf dist && \
	mkdir -p dist && \
	go build -o ssg-builder ./cmd/web/main.go && \
	./ssg-builder && \
	./tailwindcss -i ./internal/web/templates/input.css -o ./dist/styles.css --minify && \
	rm ssg-builder && \
	rm tailwindcss

# Download the tailwind css cli
setup-tailwind:
	echo "Downloading tailwind css cli..." && \
	curl -sL https://github.com/tailwindlabs/tailwindcss/releases/latest/download/tailwindcss-linux-x64 -o tailwindcss && \
	chmod +x tailwindcss

# Remove build artifacts
clean:
	rm -rf $(BUILD_DIR)
