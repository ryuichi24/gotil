PUB_PROJECT_NAME=pub
SUB_PROJECT_NAME=sub
BUILD_DIR=./dist

# Detect OS and architecture
GOOS=$(shell go env GOOS)
GOARCH=$(shell go env GOARCH)

# Build flags based on OS
ifeq ($(GOOS),linux)
    BUILD_FLAGS=-a -ldflags '-linkmode external -extldflags "-static"'
else ifeq ($(GOOS),windows)
    BUILD_FLAGS=
    CC=x86_64-w64-mingw32-gcc
else
    BUILD_FLAGS=
endif

build-pub:
	@echo "Building $(PUB_PROJECT_NAME) for $(GOOS)/$(GOARCH)..."
	@mkdir -p $(BUILD_DIR)                  # Ensure the build directory exists
	CGO_ENABLED=1 GOOS=$(GOOS) GOARCH=$(GOARCH) $(if $(CC),CC=$(CC),) go build $(BUILD_FLAGS) -o $(BUILD_DIR)/$(if $(filter windows,$(GOOS)),$(PUB_PROJECT_NAME).exe,$(PUB_PROJECT_NAME)) cmd/pub/main.go

build-sub:
	@echo "Building $(SUB_PROJECT_NAME) for $(GOOS)/$(GOARCH)..."
	@mkdir -p $(BUILD_DIR)                  # Ensure the build directory exists
	CGO_ENABLED=1 GOOS=$(GOOS) GOARCH=$(GOARCH) $(if $(CC),CC=$(CC),) go build $(BUILD_FLAGS) -o $(BUILD_DIR)/$(if $(filter windows,$(GOOS)),$(SUB_PROJECT_NAME).exe,$(SUB_PROJECT_NAME)) cmd/sub/main.go

dev-pub:
	@echo "Running $(PUB_PROJECT_NAME) pub in development mode..."
	go run cmd/pub/main.go

dev-sub:
	@echo "Running $(SUB_PROJECT_NAME) sub in development mode..."
	go run cmd/sub/main.go

clean:
	@echo "Cleaning build artifacts..."
	@rm -rf $(BUILD_DIR)

build: build-pub build-sub
	@echo "Build complete for both publisher and subscriber"

