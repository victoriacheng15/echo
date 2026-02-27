BINARY_NAME=echo
BUILD_DIR=bin

# Dynamic Nix Detection
# 1. Check if nix-shell is available
# 2. Check if we are already in a nix-shell
# 3. Check if we are NOT in GitHub Actions
USE_NIX = $(shell if command -v nix-shell >/dev/null 2>&1 && [ -z "$$IN_NIX_SHELL" ] && [ "$$GITHUB_ACTIONS" != "true" ]; then echo "yes"; else echo "no"; fi)

# Determine the target(s) for the recursive call
TARGET_GOALS = $(if $(MAKECMDGOALS),$(MAKECMDGOALS),all)

ifeq ($(USE_NIX),yes)
    NIX_RUN = nix-shell --run
    # NIX_WRAP: Re-run the entire make command inside nix-shell
    NIX_WRAP = @$(NIX_RUN) "make $(TARGET_GOALS)" && exit $$?
else
    NIX_RUN = bash -c
    NIX_WRAP = @
endif

.PHONY: all help update vet format test test-cov build clean check-env

# Default target: Run the full development lifecycle
all: update format vet test build

# Show help message
help:
	@echo "Usage: make [target]"
	@echo ""
	@echo "Available targets:"
	@echo "  all        - Run update, format, vet, test, and build"
	@echo "  help       - Show this help message"
	@echo "  update     - Run go mod tidy"
	@echo "  vet        - Run go vet"
	@echo "  format     - Run go fmt"
	@echo "  test       - Run tests"
	@echo "  test-cov   - Run tests with coverage and open HTML report"
	@echo "  build      - Build the binary under bin/"
	@echo "  clean      - Remove build artifacts"
	@echo "  check-env  - Check environment status (Nix, GitHub Actions)"

# Check the current environment (Nix, GitHub Actions)
check-env:
	$(NIX_WRAP)
	@echo "--- Environment Status ---"
	@echo "Nix Available: $(shell command -v nix-shell >/dev/null 2>&1 && echo "yes" || echo "no")"
	@echo "In Nix Shell:  $(if $(IN_NIX_SHELL),yes,no)"
	@echo "GitHub Action: $(if $(GITHUB_ACTIONS),yes,no)"
	@echo "USE_NIX:       $(USE_NIX)"
	@echo "--------------------------"

# Run go mod tidy to update dependencies
update:
	$(NIX_WRAP)
	go mod tidy

# Run go vet on all packages
vet:
	$(NIX_WRAP)
	go vet ./...

# Run go fmt on all packages
format:
	$(NIX_WRAP)
	go fmt ./...

# Run tests for all packages
test:
	$(NIX_WRAP)
	go test ./...

# Run tests with coverage and open HTML report (except in CI)
test-cov:
	$(NIX_WRAP)
	go test -coverprofile=coverage.out ./...
	@if [ "$$GITHUB_ACTIONS" != "true" ]; then \
		go tool cover -html=coverage.out; \
	fi
	rm -f coverage.out

# Build the binary under bin/
build:
	$(NIX_WRAP)
	mkdir -p $(BUILD_DIR)
	go build -o $(BUILD_DIR)/$(BINARY_NAME) ./cmd/echo

# Remove build artifacts
clean:
	rm -rf $(BUILD_DIR)
