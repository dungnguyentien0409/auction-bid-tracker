.PHONY: all mod fmt lint test coverage build run clean

APP_NAME := auction-bid-tracker
MAIN_PATH := cmd/server/main.go
BIN_DIR := bin

all: lint test build

mod:
	@echo "==> Tidy dependencies..."
	@go mod tidy

fmt:
	@echo "==> Formatting code..."
	@go fmt ./...

lint:
	@echo "==> Linting code..."
	@golangci-lint run ./...

test:
	@echo "==> Running tests..."
	@go test -v ./...

coverage:
	@echo "==> Running tests with coverage..."
	@go test -coverprofile=coverage.out ./...
	@go tool cover -html=coverage.out

build:
	@echo "==> Building $(APP_NAME)..."
	@go build -o $(BIN_DIR)/$(APP_NAME) $(MAIN_PATH)

run: build
	@echo "==> Running $(APP_NAME)..."
	@./$(BIN_DIR)/$(APP_NAME)

clean:
	@echo "==> Cleaning..."
	@rm -rf $(BIN_DIR)
	@rm -f coverage.out
