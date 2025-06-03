PROJECT_NAME=main
BUILD_DIR=./dist

# Detect OS and architecture
GOOS=$(shell go env GOOS)
GOARCH=$(shell go env GOARCH)

# Build flags based on OS
ifeq ($(GOOS),linux)
    BUILD_FLAGS=
else ifeq ($(GOOS),windows)
    BUILD_FLAGS=
    CC=
else
    BUILD_FLAGS=
endif

dev:
	@echo "Running development server for $(PROJECT_NAME)..."
	@air

build: build-writer build-reader

build-writer:
	@echo "Building $(PROJECT_NAME)..."
	@go build -o $(BUILD_DIR)/writer cmd/writer/main.go
	
build-reader:
	@echo "Building $(PROJECT_NAME)..."
	@go build -o $(BUILD_DIR)/reader cmd/reader/main.go

clean:
	@echo "Cleaning build artifacts..."
	@rm -rf $(BUILD_DIR)

# 
.PHONY: dev build clean