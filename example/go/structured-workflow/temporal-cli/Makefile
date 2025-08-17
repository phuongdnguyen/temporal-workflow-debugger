# Temporal CLI Makefile

.PHONY: help build clean install run-worker run-workflow test fmt vet deps

# Default target
help:
	@echo "Temporal CLI - Available targets:"
	@echo ""
	@echo "  build        - Build the temporal-cli binary"
	@echo "  install      - Install dependencies"
	@echo "  clean        - Clean build artifacts"
	@echo "  run-worker   - Start the Temporal worker"
	@echo "  run-workflow - Run a workflow execution"
	@echo "  test         - Run tests"
	@echo "  fmt          - Format Go code"
	@echo "  vet          - Run go vet"
	@echo "  deps         - Download and tidy dependencies"
	@echo "  help         - Show this help message"

# Build the binary
build:
	@echo "Building temporal-cli..."
	go build -o temporal-cli .

# Install/download dependencies
install deps:
	@echo "Installing dependencies..."
	go mod download
	go mod tidy

# Clean build artifacts
clean:
	@echo "Cleaning build artifacts..."
	rm -f temporal-cli
	go clean

# Start the Temporal worker
run-worker start-worker:
	@echo "Starting Temporal worker..."
	go run . start

# Run a workflow execution
run-workflow start-workflow:
	@echo "Running workflow..."
	go run . run

# Run tests
test:
	@echo "Running tests..."
	go test -v ./...

# Format Go code
fmt:
	@echo "Formatting Go code..."
	go fmt ./...

# Run go vet
vet:
	@echo "Running go vet..."
	go vet ./...

# Development workflow - format, vet, and build
dev: fmt vet build
	@echo "Development build complete"

# Quick check - format, vet, and test
check: fmt vet test
	@echo "Code check complete"
