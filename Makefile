.PHONY: all mod fmt lint unit integration coverage build run clean benchmark load-test

APP_NAME := auction-bid-tracker
MAIN_PATH := cmd/server/main.go
BIN_DIR := bin

all: lint unit build

mod:
	@echo "==> Tidy dependencies..."
	@go mod tidy

fmt:
	@echo "==> Formatting code..."
	@go fmt ./...

lint:
	@echo "==> Linting code..."
	@golangci-lint run ./...

unit:
	@echo "==> Running unit tests..."
	@go test -v ./...

integration:
	@echo "==> Running integration tests..."
	@go test -v -tags=integration ./tests/...

coverage:
	@echo "==> Running tests with coverage..."
	@go test -coverprofile=coverage.out ./...
	@go tool cover -html=coverage.out

build:
	@echo "==> Building $(APP_NAME)..."
	@go build -o $(BIN_DIR)/$(APP_NAME) $(MAIN_PATH)

run: build
	@echo "==> Running $(APP_NAME)..."
	@APP_ENV=development ./$(BIN_DIR)/$(APP_NAME)

clean:
	@echo "==> Cleaning..."
	@rm -rf $(BIN_DIR)
	@rm -f coverage.out

benchmark:
	@echo "==> Running benchmarks..."
	@go test -bench=. -benchmem ./...

load-test: build
	@echo "==> Cleaning up port 8080..."
	@lsof -ti:8080 | xargs kill -9 || true
	@sleep 1
	@echo "==> Starting fresh server for load test..."
	@APP_ENV=development ./$(BIN_DIR)/$(APP_NAME) & \
	SERVER_PID=$$!; \
	sleep 2; \
	echo "==> Running load test..."; \
	go run cmd/loadtest/main.go -workers=200 -duration=10s; \
	echo "==> Stopping server..."; \
	kill $$SERVER_PID || true
