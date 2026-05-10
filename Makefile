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
	@APP_ENV=stress ./$(BIN_DIR)/$(APP_NAME) & \
	SERVER_PID=$$!; \
	sleep 2; \
	echo "==> Running load test..."; \
	go run cmd/loadtest/main.go -workers=200 -duration=10s; \
	echo "==> Stopping server..."; \
	kill $$SERVER_PID || true

contention-test: build
	@echo "==> SCENARIO: Hot Auction (1 Item)"
	@echo "==> DESCRIPTION: Maximum lock contention on a single resource. Tests the extreme limits of the atomic sync path."
	@lsof -ti:8080 | xargs kill -9 || true; sleep 1
	@APP_ENV=stress ./$(BIN_DIR)/$(APP_NAME) > /dev/null 2>&1 & \
	SERVER_PID=$$!; sleep 2; \
	go run cmd/loadtest/main.go -workers=200 -duration=10s -hot; \
	kill $$SERVER_PID || true

test-trending: build
	@echo "==> SCENARIO: Trending Auctions (10 Items)"
	@echo "==> DESCRIPTION: High contention across a small set of popular items. Tests sharded lock efficiency."
	@lsof -ti:8080 | xargs kill -9 || true; sleep 1
	@APP_ENV=stress ./$(BIN_DIR)/$(APP_NAME) > /dev/null 2>&1 & \
	SERVER_PID=$$!; sleep 2; \
	go run cmd/loadtest/main.go -workers=200 -duration=10s -items=10; \
	kill $$SERVER_PID || true

test-distributed: build
	@echo "==> SCENARIO: Distributed Load (1000 Items)"
	@echo "==> DESCRIPTION: Wide distribution with low contention. Tests the system's peak throughput capacity."
	@lsof -ti:8080 | xargs kill -9 || true; sleep 1
	@APP_ENV=stress ./$(BIN_DIR)/$(APP_NAME) > /dev/null 2>&1 & \
	SERVER_PID=$$!; sleep 2; \
	go run cmd/loadtest/main.go -workers=200 -duration=10s -items=1000; \
	kill $$SERVER_PID || true

test-zipf: build
	@echo "==> SCENARIO: Skewed Load (Zipfian, 1000 Items)"
	@echo "==> DESCRIPTION: 80/20 distribution pattern. Most realistic simulation of real-world auction behavior."
	@lsof -ti:8080 | xargs kill -9 || true; sleep 1
	@APP_ENV=stress ./$(BIN_DIR)/$(APP_NAME) > /dev/null 2>&1 & \
	SERVER_PID=$$!; sleep 2; \
	go run cmd/loadtest/main.go -workers=200 -duration=10s -items=1000 -dist=zipf; \
	kill $$SERVER_PID || true

stress-matrix: build
	@echo "=========================================================="
	@echo "      AUCTION BID TRACKER - PERFORMANCE MATRIX            "
	@echo "=========================================================="
	@echo ""
	@$(MAKE) contention-test | grep -E "SCENARIO|Throughput|Latency|Time Taken|Total Requests|Successful|Failed"
	@echo "----------------------------------------------------------"
	@$(MAKE) test-trending | grep -E "SCENARIO|Throughput|Latency|Time Taken|Total Requests|Successful|Failed"
	@echo "----------------------------------------------------------"
	@$(MAKE) test-distributed | grep -E "SCENARIO|Throughput|Latency|Time Taken|Total Requests|Successful|Failed"
	@echo "----------------------------------------------------------"
	@$(MAKE) test-zipf | grep -E "SCENARIO|Throughput|Latency|Time Taken|Total Requests|Successful|Failed"
	@echo ""
	@echo "=========================================================="
	@echo "MATRIX TEST COMPLETED"
	@echo "=========================================================="

docker-build:
	@echo "==> Building Docker image..."
	@docker build -t auction-tracker .

docker-run: docker-build
	@echo "==> Running Docker container on port 8080..."
	@docker run -p 8080:8080 auction-tracker
