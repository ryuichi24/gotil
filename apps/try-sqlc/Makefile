PROJECT_NAME=try-sqlc
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

build:
	@echo "Building $(PROJECT_NAME)..."
	@go build -o $(BUILD_DIR)/$(PROJECT_NAME) cmd/try-sqlc/main.go
	
clean:
	@echo "Cleaning build artifacts..."
	@rm -rf $(BUILD_DIR)

# Database configuration
DB_FILE ?= internal/db/database.db

# Database commands
gensqlc:
	@echo "Generating SQLC files..."
	@sqlc generate

name ?= "default"
genmgt:
	@echo "Generating management files..."
	@goose create $(name) sql -dir internal/db/sql/migrations

db ?= "./test.db"
up:
	@goose sqlite3 $(db) -dir internal/db/sql/migrations up

down:
	@goose sqlite3 $(db) -dir internal/db/sql/migrations down

# 
.PHONY: dev build clean genmgt db-init migrate-up migrate-down