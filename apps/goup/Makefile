PROJECT_NAME=goup
BUILD_DIR=./dist

build:
	@echo "Building $(PROJECT_NAME)..."
	@mkdir -p $(BUILD_DIR)                  # Ensure the build directory exists
	go build -o $(BUILD_DIR)/$(PROJECT_NAME) cmd/server/main.go

dev:
	@echo "Running $(PROJECT_NAME) server in development mode..."
	go run cmd/server/main.go

clean:
	@echo "Cleaning build artifacts..."
	@rm -rf $(BUILD_DIR)