# Makefile for GOReilly project: lint, fmt, build, test

.PHONY: all fmt lint build test clean

# Default to all
all: fmt lint build test

# Format all Go files in the module
fmt:
	go fmt ./...

# Run go vet and golint on all packages
lint:
	go vet ./...
	@which golint >/dev/null 2>&1 || (echo "Installing golint..."; go install golang.org/x/lint/golint@latest)
	@golint ./...

# Build all main packages recursively
build:
	go build ./...

# Run all tests (if any) in verbose mode
test:
	go test -v ./...

# Remove binary outputs (assuming default behavior)
clean:
	find . -type f -name "*.out" -delete
	go clean

# Help info
help:
	@echo "Usage: make [target]"
	@echo ""
	@echo "Targets:"
	@echo "  all     - Run fmt, lint, build, and test"
	@echo "  fmt     - Format all Go code"
	@echo "  lint    - Run go vet and golint"
	@echo "  build   - Build all packages"
	@echo "  test    - Run all tests"
	@echo "  clean   - Remove build/test artifacts"
	@echo "  help    - This help message"
