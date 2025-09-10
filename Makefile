# Makefile for Proxy Forward Server

# Variables
BINARY_NAME=proxy-forward
BIN_DIR=bin
SOURCE_FILE=main.go
BUILD_FLAGS=-ldflags="-s -w"

# Build the application
build:
	@echo "Building $(BINARY_NAME)..."
	@mkdir -p $(BIN_DIR)
	go build $(BUILD_FLAGS) -o $(BIN_DIR)/$(BINARY_NAME) $(SOURCE_FILE)
	@echo "Built successfully: $(BIN_DIR)/$(BINARY_NAME)"

# Build for different platforms
build-linux:
	@echo "Building for Linux..."
	@mkdir -p $(BIN_DIR)
	GOOS=linux GOARCH=amd64 go build $(BUILD_FLAGS) -o $(BIN_DIR)/$(BINARY_NAME)-linux-amd64 $(SOURCE_FILE)

build-windows:
	@echo "Building for Windows..."
	@mkdir -p $(BIN_DIR)
	GOOS=windows GOARCH=amd64 go build $(BUILD_FLAGS) -o $(BIN_DIR)/$(BINARY_NAME)-windows-amd64.exe $(SOURCE_FILE)

build-mac:
	@echo "Building for macOS..."
	@mkdir -p $(BIN_DIR)
	GOOS=darwin GOARCH=amd64 go build $(BUILD_FLAGS) -o $(BIN_DIR)/$(BINARY_NAME)-darwin-amd64 $(SOURCE_FILE)

# Build for all platforms
build-all: build-linux build-windows build-mac
	@echo "Built for all platforms"

# Run the application
run: build
	@echo "Starting proxy server..."
	./$(BIN_DIR)/$(BINARY_NAME)

# Clean build artifacts
clean:
	@echo "Cleaning build artifacts..."
	rm -rf $(BIN_DIR)
	@echo "Cleaned"

# Install dependencies
deps:
	@echo "Installing dependencies..."
	go mod tidy
	go mod download

# Test the application
test:
	@echo "Running tests..."
	go test ./...

# Show help
help:
	@echo "Available commands:"
	@echo "  build        - Build the application"
	@echo "  build-linux  - Build for Linux"
	@echo "  build-windows- Build for Windows"
	@echo "  build-mac    - Build for macOS"
	@echo "  build-all    - Build for all platforms"
	@echo "  run          - Build and run the application"
	@echo "  clean        - Clean build artifacts"
	@echo "  deps         - Install dependencies"
	@echo "  test         - Run tests"
	@echo "  help         - Show this help message"

.PHONY: build build-linux build-windows build-mac build-all run clean deps test help
