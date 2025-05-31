PROJECT_NAME=curd-serve
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
	@go run cmd/server/main.go

build:
	@echo "Building $(PROJECT_NAME)..."
	
clean:
	@echo "Cleaning build artifacts..."
	@rm -rf $(BUILD_DIR)

