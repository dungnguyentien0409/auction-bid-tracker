.PHONY: all build run test clean fmt mod coverage

APP_NAME := auction-bid-tracker
MAIN_PATH := cmd/server/main.go
BIN_DIR := bin

all: build

build:
	@echo "==> Building $(APP_NAME)..."
	@go build -o $(BIN_DIR)/$(APP_NAME) $(MAIN_PATH)

run: build
	@echo "==> Running $(APP_NAME)..."
	@./$(BIN_DIR)/$(APP_NAME)

test:
	@echo "==> Running tests..."
	@go test -v ./...

clean:
	@echo "==> Cleaning..."
	@rm -rf $(BIN_DIR)

fmt:
	@echo "==> Formatting code..."
	@go fmt ./...

mod:
	@echo "==> Tidy dependencies..."
	@go mod tidy

coverage:
	@echo "==> Running tests with coverage..."
	@go test -coverprofile=coverage.out ./...
	@go tool cover -html=coverage.out
