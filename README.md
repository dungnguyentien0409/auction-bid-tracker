# Auction Bid Tracker

A high-performance bidding engine written in Go, focused on low-latency data consistency and high-concurrency throughput. 

This system was built to handle extreme load (100k+ RPS) on a single node by utilizing fine-grained locking and clean architectural principles.

## Performance Results

The following metrics were captured during stress tests on an Apple M1 Pro (10-core):

- **Micro-Benchmark (Core Engine)**: 
    - Parallel (Multiple Items): **~79 ns/op** (12M+ ops/sec).
    - Contention (Single Item): **~177 ns/op** (5.6M+ ops/sec).
- **Macro-Load Test (API)**: **~72,000+ RPS** (with Recovery middleware enabled).
- **Hot Auction Scenario**: **~70,000+ RPS** (All traffic concentrated on a **single item** to verify lock contention handling).
- **Average Latency**: **~2.2ms** average response time under heavy concurrent load.
- **Stability**: **99.99%+ success rate** (Zero logic failures, minimal network jitter observed).

## System Architecture

```text
       [ External ]         |          [ Internal / Core Logic ]
                            |
    (HTTP / JSON)           |        (Golang Domain Interfaces)
  Clients --(POST/GET)--> [ API Handler ] <---- [ Tracker Interface ]
                            |     |                   ^
                            |     |                   |
                            | [ Recovery ]     [ Bid Service ]
                            |                         |
                            |                         v
                            |               [ Repository Interface ]
                            |                         |
                            |                [ Memory Repository ]
                            |               /         |         \
                            |      [ Lock:Item1 ] [ Lock:Item2 ] [ Lock:Item3 ]
                            |            |              |              |
                            |      [ Bids Data ]  [ Bids Data ]  [ Bids Data ]
```

## Technical Design Decisions

### 1. Concurrency & Performance (Fine-Grained Locking)
To achieve high throughput without global blocking, I implemented a **Fine-Grained Locking** strategy:
- **`sync.Map`**: Used as the primary registry for auction items to allow lock-free reads for most operations.
- **Per-Item `RWMutex`**: Each auction item is encapsulated in an `itemRecord` which contains its own `sync.RWMutex`. 
- **The Result**: Parallel processing of bids for different items, eliminating lock contention. Even in the "Hot Auction" (single item) scenario, the system maintains high throughput due to the optimized locking path.

### 2. Dependency Injection & Scalability
The system is built on **Interface-Driven Design**:
- `Tracker` and `Repository` interfaces decouple the business logic from the infrastructure.
- This allows for easy swapping of implementations—for instance, migrating from in-memory storage to a persistent database like PostgreSQL requires minimal changes to the bootstrap logic in `main.go`.
- This pattern also facilitated achieving **100% test coverage** by allowing easy mocking of components.

### 3. Fault Tolerance
Stability is prioritized alongside performance:
- **Recovery Middleware**: Every request is wrapped in a recovery defer block. This ensures that an unexpected panic in any handler won't crash the entire server, maintaining 24/7 availability.

### 4. Production Readiness
- **Structured Logging**: Using `log/slog` for environment-aware logging.
- **Graceful Shutdown**: The server handles OS signals (SIGINT/SIGTERM) to finish active requests before exiting.

## Testing & Verification

The project maintains **100% unit test coverage** for all internal logic.

- **Unit & Integration Tests**: `make test`
- **Coverage Report**: `make coverage` (Verified 100% on core packages)
- **Performance Benchmarks**: `make benchmark`
- **End-to-End Load Test**: `make load-test` 
- **Lock Contention Test**: `make contention-test` (Single item stress test)

## Getting Started

### Prerequisites
- Go 1.22+

### Instructions
1. **Run the server**:
   ```bash
   make run
   ```
2. **Run tests**:
   ```bash
   make test
   make coverage
   ```
3. **Docker Support**:
   - Run via Docker: `make docker-run` (auto-builds image)

4. **Performance verification**:
   - `make benchmark` (Core logic speed)
   - `make load-test` (API throughput)
   - `make contention-test` (Single item contention test)

## Project Layout
- `cmd/`: Entry points for the server and load tester.
- `internal/api/`: HTTP layer and middlewares.
- `internal/domain/`: Core entities and repository contracts.
- `internal/repository/`: Thread-safe data structures with fine-grained locking.
- `internal/service/`: Business logic orchestration.