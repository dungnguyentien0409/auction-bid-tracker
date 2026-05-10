.PHONY: all help mod fmt lint unit integration coverage build run clean benchmark load-test stress-matrix docker-up docker-down

help: ## Display this help screen
	@echo "Usage: make <target> [APP_ENV=<env>] [REPO_TYPE=<repo>]"
	@echo ""
	@echo "Targets:"
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-20s\033[0m %s\n", $$1, $$2}'
	@echo ""
	@echo "Envs:    development, stress, production"
	@echo "Repos:   memory, redis"

APP_NAME := auction-bid-tracker
MAIN_PATH := cmd/server/main.go
BIN_DIR := bin

# Load Test Parameters
DURATION ?= 10s
WORKERS ?= 200

# Mandatory Variables Validation
validate-env:
ifndef APP_ENV
	@echo "\033[1;31m[!] Missing APP_ENV\033[0m (development | stress | production)"
	@echo "    Usage: \033[1;36mmake $(MAKECMDGOALS) APP_ENV=development REPO_TYPE=memory\033[0m"
	@exit 1
endif
ifndef REPO_TYPE
	@echo "\033[1;31m[!] Missing REPO_TYPE\033[0m (memory | redis)"
	@echo "    Usage: \033[1;36mmake $(MAKECMDGOALS) REPO_TYPE=memory\033[0m"
	@exit 1
endif

all: validate-env lint unit build

mod:
	@echo "==> Tidy dependencies..."
	@go mod tidy

fmt:
	@echo "==> Formatting code..."
	@go fmt ./...

lint: ## Run golangci-lint
	@echo "==> Linting code..."
	@golangci-lint run ./...

unit: ## Run unit tests
	@echo "==> Running unit tests..."
	@APP_ENV=development go test -v ./...

integration: ## Run integration tests for all backends
	@echo "==> Running integration tests (All Backends)..."
	@$(MAKE) do-integration REPO_TYPE=memory APP_ENV=development
	@$(MAKE) do-integration REPO_TYPE=redis APP_ENV=development

do-integration: validate-env
	@printf "\033[1;35m\n============================================================\n  🔗 INTEGRATION TEST — ENV: $(APP_ENV) | BACKEND: $(REPO_TYPE)\n============================================================\033[0m\n"
	@REDIS_STARTED=0; \
	if [ "$(REPO_TYPE)" = "redis" ] && ! nc -z localhost 6379 > /dev/null 2>&1; then \
		echo "    ==> Starting ephemeral Redis for integration tests..."; \
		docker rm -f redis-integration > /dev/null 2>&1 || true; \
		docker run -d --name redis-integration -p 6379:6379 redis:alpine > /dev/null; \
		REDIS_STARTED=1; \
		until nc -z localhost 6379 > /dev/null 2>&1; do sleep 1; done; \
	fi; \
	APP_ENV=$(APP_ENV) REPO_TYPE=$(REPO_TYPE) go test -v ./tests/... ./internal/... -tags=integration; \
	TEST_EXIT=$$?; \
	if [ "$$REDIS_STARTED" = "1" ]; then \
		echo "    ==> Stopping ephemeral Redis..."; \
		docker rm -f redis-integration > /dev/null 2>&1 || true; \
	fi; \
	exit $$TEST_EXIT

coverage:
	@echo "==> Running tests with coverage..."
	@go test -coverprofile=coverage.out ./...
	@go tool cover -html=coverage.out

build: validate-env ## Build the server binary
	@echo "==> Building $(APP_NAME)..."
	@go build -o $(BIN_DIR)/$(APP_NAME) $(MAIN_PATH)

run: build ## Run the standalone server locally (Use SEED=true to pre-populate)
	@if [ "$(SEED)" = "true" ]; then \
		echo "==> Starting server in background for seeding..."; \
		APP_ENV=development REPO_TYPE=memory ./$(BIN_DIR)/$(APP_NAME) > /dev/null 2>&1 & \
		SERVER_PID=$$!; \
		$(MAKE) wait-server; \
		$(MAKE) seed; \
		echo "==> Seeding complete. Server is running (PID: $$SERVER_PID)."; \
		wait $$SERVER_PID; \
	else \
		APP_ENV=development REPO_TYPE=memory ./$(BIN_DIR)/$(APP_NAME); \
	fi

wait-server:
	@echo "==> Waiting for server to be ready (Timeout: 20s)..."
	@for i in $$(seq 1 20); do \
		if curl -s http://localhost:8080/health > /dev/null; then \
			echo "==> Server is UP!"; \
			exit 0; \
		fi; \
		echo "    Attempt $$i/20: Server not ready, retrying in 1s..."; \
		sleep 1; \
	done; \
	echo "==> ERROR: Server failed to start"; \
	exit 1

seed: ## Seed sample data via public API (Internal use)
	@echo "==> SEEDING: Injecting mock data via API..."
	@curl -s -X POST http://localhost:8080/bids -H "Content-Type: application/json" -d '{"item_id": "macbook-m3", "user_id": "user-1", "amount": 2000.0}' > /dev/null
	@curl -s -X POST http://localhost:8080/bids -H "Content-Type: application/json" -d '{"item_id": "macbook-m3", "user_id": "user-2", "amount": 2100.0}' > /dev/null
	@curl -s -X POST http://localhost:8080/bids -H "Content-Type: application/json" -d '{"item_id": "macbook-m3", "user_id": "user-3", "amount": 2500.0}' > /dev/null
	@curl -s -X POST http://localhost:8080/bids -H "Content-Type: application/json" -d '{"item_id": "leica-m6", "user_id": "user-1", "amount": 3000.0}' > /dev/null
	@echo "==> SEEDING: Mock data injected successfully."

docker-up: ## Start the distributed system (Use SEED=true to pre-populate)
	@echo "==> Starting distributed system..."
	@docker-compose up -d --build
	@if [ "$(SEED)" = "true" ]; then \
		$(MAKE) wait-server; \
		$(MAKE) seed; \
	fi
	@echo "==> Cluster is UP. Run 'docker-compose logs -f' to follow logs."

docker-down: ## Stop the distributed system
	@echo "==> Stopping distributed system..."
	@docker-compose down -v

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
	@echo "==> Starting fresh server for load test..."
	@APP_ENV=stress REPO_TYPE=$(REPO_TYPE) ./$(BIN_DIR)/$(APP_NAME) > /dev/null 2>&1 & \
	SERVER_PID=$$!; \
	$(MAKE) wait-server; \
	echo "==> Running load test..."; \
	go run cmd/loadtest/main.go -workers=$(WORKERS) -duration=$(DURATION); \
	echo "==> Stopping server..."; \
	kill $$SERVER_PID || true

contention-test: build
	@echo "==> SCENARIO: Hot Auction (1 Item)"
	@echo "==> DESCRIPTION: Maximum lock contention on a single resource. Tests the extreme limits of the atomic sync path."
	@lsof -ti:8080 | xargs kill -9 || true
	@APP_ENV=stress REPO_TYPE=$(REPO_TYPE) ./$(BIN_DIR)/$(APP_NAME) > /dev/null 2>&1 & \
	SERVER_PID=$$!; \
	$(MAKE) wait-server; \
	go run cmd/loadtest/main.go -workers=$(WORKERS) -duration=$(DURATION) -hot; \
	kill $$SERVER_PID || true

test-trending: build
	@echo "==> SCENARIO: Trending Auctions (10 Items)"
	@echo "==> DESCRIPTION: High contention across a small set of popular items. Tests sharded lock efficiency."
	@lsof -ti:8080 | xargs kill -9 || true
	@APP_ENV=stress REPO_TYPE=$(REPO_TYPE) ./$(BIN_DIR)/$(APP_NAME) > /dev/null 2>&1 & \
	SERVER_PID=$$!; \
	$(MAKE) wait-server; \
	go run cmd/loadtest/main.go -workers=$(WORKERS) -duration=$(DURATION) -items=10; \
	kill $$SERVER_PID || true

test-distributed: build
	@echo "==> SCENARIO: Distributed Load (1000 Items)"
	@echo "==> DESCRIPTION: Wide distribution with low contention. Tests the system's peak throughput capacity."
	@lsof -ti:8080 | xargs kill -9 || true
	@APP_ENV=stress REPO_TYPE=$(REPO_TYPE) ./$(BIN_DIR)/$(APP_NAME) > /dev/null 2>&1 & \
	SERVER_PID=$$!; \
	$(MAKE) wait-server; \
	go run cmd/loadtest/main.go -workers=$(WORKERS) -duration=$(DURATION) -items=1000; \
	kill $$SERVER_PID || true

test-zipf: build
	@echo "==> SCENARIO: Skewed Load (Zipfian, 1000 Items)"
	@echo "==> DESCRIPTION: 80/20 distribution pattern. Most realistic simulation of real-world auction behavior."
	@lsof -ti:8080 | xargs kill -9 || true
	@APP_ENV=stress REPO_TYPE=$(REPO_TYPE) ./$(BIN_DIR)/$(APP_NAME) > /dev/null 2>&1 & \
	SERVER_PID=$$!; \
	$(MAKE) wait-server; \
	go run cmd/loadtest/main.go -workers=$(WORKERS) -duration=$(DURATION) -items=1000 -dist=zipf; \
	kill $$SERVER_PID || true

stress-matrix: ## Run the full performance matrix (APP_ENV=stress)
	@$(MAKE) build APP_ENV=stress REPO_TYPE=$(REPO_TYPE)
	@REDIS_STARTED=0; \
	if [ "$(REPO_TYPE)" = "redis" ]; then \
		if ! nc -z localhost 6379 > /dev/null 2>&1; then \
			echo "==> Starting Redis container for stress test..."; \
			docker rm -f redis-stress > /dev/null 2>&1 || true; \
			docker run -d --name redis-stress -p 6379:6379 redis:alpine; \
			REDIS_STARTED=1; \
			echo "==> Waiting for Redis to be ready..."; \
			until nc -z localhost 6379; do sleep 1; done; \
		fi; \
	fi; \
	echo "=========================================================="; \
	echo "      AUCTION BID TRACKER - PERFORMANCE MATRIX            "; \
	echo "      ENV:     stress                                     "; \
	echo "      BACKEND: $(REPO_TYPE)                               "; \
	echo "=========================================================="; \
	echo ""; \
	$(MAKE) contention-test APP_ENV=stress REPO_TYPE=$(REPO_TYPE) | grep -E "SCENARIO|Throughput|Latency|Time Taken|Total Requests|Successful|Failed"; \
	echo "----------------------------------------------------------"; \
	$(MAKE) test-trending APP_ENV=stress REPO_TYPE=$(REPO_TYPE) | grep -E "SCENARIO|Throughput|Latency|Time Taken|Total Requests|Successful|Failed"; \
	echo "----------------------------------------------------------"; \
	$(MAKE) test-distributed APP_ENV=stress REPO_TYPE=$(REPO_TYPE) | grep -E "SCENARIO|Throughput|Latency|Time Taken|Total Requests|Successful|Failed"; \
	echo "----------------------------------------------------------"; \
	$(MAKE) test-zipf APP_ENV=stress REPO_TYPE=$(REPO_TYPE) | grep -E "SCENARIO|Throughput|Latency|Time Taken|Total Requests|Successful|Failed"; \
	echo ""; \
	echo "=========================================================="; \
	echo "MATRIX TEST COMPLETED"; \
	echo "=========================================================="; \
	if [ "$$REDIS_STARTED" = "1" ]; then \
		echo "==> Stopping temporary Redis container..."; \
		docker rm -f redis-stress > /dev/null 2>&1 || true; \
	fi

stress-compare: ## Run Memory vs Redis deep comparison (Default: DURATION=5s)
	@echo "==> Starting Deep Comparative Stress Test (Memory vs Redis)..."
	@echo "    Duration per scenario: $(if $(DURATION),$(DURATION),5s)"
	@echo "    This will take about 1 minute. Please wait..."
	@$(MAKE) stress-matrix REPO_TYPE=memory DURATION=$(if $(DURATION),$(DURATION),5s) > stress_memory.log 2>&1
	@$(MAKE) stress-matrix REPO_TYPE=redis DURATION=$(if $(DURATION),$(DURATION),5s) > stress_redis.log 2>&1
	@echo ""
	@echo "==========================================================================================="
	@echo "                       AUCTION SYSTEM FULL PERFORMANCE AUDIT                               "
	@echo "==========================================================================================="
	@printf "| %-22s | %-8s | %-12s | %-10s | %-10s | %-5s |\n" "Scenario" "Backend" "RPS" "Latency" "Success" "Fail"
	@echo "|------------------------|----------|--------------|------------|------------|-------|"
	@M_RPS=$$(grep "Hot Auction" -A 10 stress_memory.log | grep "Throughput" | awk '{print $$3}'); \
	 M_LAT=$$(grep "Hot Auction" -A 10 stress_memory.log | grep "Average Latency" | awk '{print $$3,$$4}'); \
	 M_SUC=$$(grep "Hot Auction" -A 10 stress_memory.log | grep "Total Requests" | awk '{print $$3}'); \
	 M_FAIL=$$(grep "Hot Auction" -A 10 stress_memory.log | grep "Failed Requests" | awk '{print $$3}'); \
	 R_RPS=$$(grep "Hot Auction" -A 10 stress_redis.log | grep "Throughput" | awk '{print $$3}'); \
	 R_LAT=$$(grep "Hot Auction" -A 10 stress_redis.log | grep "Average Latency" | awk '{print $$3,$$4}'); \
	 R_SUC=$$(grep "Hot Auction" -A 10 stress_redis.log | grep "Total Requests" | awk '{print $$3}'); \
	 R_FAIL=$$(grep "Hot Auction" -A 10 stress_redis.log | grep "Failed Requests" | awk '{print $$3}'); \
	 printf "| %-22s | %-8s | %-12s | %-10s | %-10s | %-5s |\n" "Hot Auction (1 Item)" "Memory" "$$M_RPS" "$$M_LAT" "$$M_SUC" "$$M_FAIL"; \
	 printf "| %-22s | %-8s | %-12s | %-10s | %-10s | %-5s |\n" "" "Redis" "$$R_RPS" "$$R_LAT" "$$R_SUC" "$$R_FAIL"
	@echo "|------------------------|----------|--------------|------------|------------|-------|"
	@M_RPS=$$(grep "Trending Auctions" -A 10 stress_memory.log | grep "Throughput" | awk '{print $$3}'); \
	 M_LAT=$$(grep "Trending Auctions" -A 10 stress_memory.log | grep "Average Latency" | awk '{print $$3,$$4}'); \
	 M_SUC=$$(grep "Trending Auctions" -A 10 stress_memory.log | grep "Total Requests" | awk '{print $$3}'); \
	 M_FAIL=$$(grep "Trending Auctions" -A 10 stress_memory.log | grep "Failed Requests" | awk '{print $$3}'); \
	 R_RPS=$$(grep "Trending Auctions" -A 10 stress_redis.log | grep "Throughput" | awk '{print $$3}'); \
	 R_LAT=$$(grep "Trending Auctions" -A 10 stress_redis.log | grep "Average Latency" | awk '{print $$3,$$4}'); \
	 R_SUC=$$(grep "Trending Auctions" -A 10 stress_redis.log | grep "Total Requests" | awk '{print $$3}'); \
	 R_FAIL=$$(grep "Trending Auctions" -A 10 stress_redis.log | grep "Failed Requests" | awk '{print $$3}'); \
	 printf "| %-22s | %-8s | %-12s | %-10s | %-10s | %-5s |\n" "Trending (10 Items)" "Memory" "$$M_RPS" "$$M_LAT" "$$M_SUC" "$$M_FAIL"; \
	 printf "| %-22s | %-8s | %-12s | %-10s | %-10s | %-5s |\n" "" "Redis" "$$R_RPS" "$$R_LAT" "$$R_SUC" "$$R_FAIL"
	@echo "|------------------------|----------|--------------|------------|------------|-------|"
	@M_RPS=$$(grep "Distributed Load" -A 10 stress_memory.log | grep "Throughput" | awk '{print $$3}'); \
	 M_LAT=$$(grep "Distributed Load" -A 10 stress_memory.log | grep "Average Latency" | awk '{print $$3,$$4}'); \
	 M_SUC=$$(grep "Distributed Load" -A 10 stress_memory.log | grep "Total Requests" | awk '{print $$3}'); \
	 M_FAIL=$$(grep "Distributed Load" -A 10 stress_memory.log | grep "Failed Requests" | awk '{print $$3}'); \
	 R_RPS=$$(grep "Distributed Load" -A 10 stress_redis.log | grep "Throughput" | awk '{print $$3}'); \
	 R_LAT=$$(grep "Distributed Load" -A 10 stress_redis.log | grep "Average Latency" | awk '{print $$3,$$4}'); \
	 R_SUC=$$(grep "Distributed Load" -A 10 stress_redis.log | grep "Total Requests" | awk '{print $$3}'); \
	 R_FAIL=$$(grep "Distributed Load" -A 10 stress_redis.log | grep "Failed Requests" | awk '{print $$3}'); \
	 printf "| %-22s | %-8s | %-12s | %-10s | %-10s | %-5s |\n" "Distributed (1000)" "Memory" "$$M_RPS" "$$M_LAT" "$$M_SUC" "$$M_FAIL"; \
	 printf "| %-22s | %-8s | %-12s | %-10s | %-10s | %-5s |\n" "" "Redis" "$$R_RPS" "$$R_LAT" "$$R_SUC" "$$R_FAIL"
	@echo "|------------------------|----------|--------------|------------|------------|-------|"
	@M_RPS=$$(grep "Skewed Load" -A 10 stress_memory.log | grep "Throughput" | awk '{print $$3}'); \
	 M_LAT=$$(grep "Skewed Load" -A 10 stress_memory.log | grep "Average Latency" | awk '{print $$3,$$4}'); \
	 M_SUC=$$(grep "Skewed Load" -A 10 stress_memory.log | grep "Total Requests" | awk '{print $$3}'); \
	 M_FAIL=$$(grep "Skewed Load" -A 10 stress_memory.log | grep "Failed Requests" | awk '{print $$3}'); \
	 R_RPS=$$(grep "Skewed Load" -A 10 stress_redis.log | grep "Throughput" | awk '{print $$3}'); \
	 R_LAT=$$(grep "Skewed Load" -A 10 stress_redis.log | grep "Average Latency" | awk '{print $$3,$$4}'); \
	 R_SUC=$$(grep "Skewed Load" -A 10 stress_redis.log | grep "Total Requests" | awk '{print $$3}'); \
	 R_FAIL=$$(grep "Skewed Load" -A 10 stress_redis.log | grep "Failed Requests" | awk '{print $$3}'); \
	 printf "| %-22s | %-8s | %-12s | %-10s | %-10s | %-5s |\n" "Skewed (Zipfian)" "Memory" "$$M_RPS" "$$M_LAT" "$$M_SUC" "$$M_FAIL"; \
	 printf "| %-22s | %-8s | %-12s | %-10s | %-10s | %-5s |\n" "" "Redis" "$$R_RPS" "$$R_LAT" "$$R_SUC" "$$R_FAIL"
	@echo "==========================================================================================="
	@rm -f stress_memory.log stress_redis.log
	@echo "Deep Audit complete. Memory exhibits higher throughput and lower latency as expected."


docker-build:
	@echo "==> Building Docker image..."
	@docker build -t auction-tracker .

docker-run: docker-build
	@echo "==> Running Docker container on port 8080 (REPO_TYPE=$(REPO_TYPE))..."
	@docker run -p 8080:8080 -e REPO_TYPE=$(REPO_TYPE) auction-tracker
