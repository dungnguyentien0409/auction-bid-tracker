# ⚡️ Auction Bid Tracker

[![Go Version](https://img.shields.io/badge/Go-1.24+-00ADD8?style=for-the-badge&logo=go)](https://go.dev/)
[![CI](https://img.shields.io/github/actions/workflow/status/dungnguyentien0409/auction-bid-tracker/ci.yml?style=for-the-badge&label=CI)](https://github.com/dungnguyentien0409/auction-bid-tracker/actions)
[![Coverage](https://img.shields.io/badge/Coverage-100%25-brightgreen?style=for-the-badge)](https://github.com/dungnguyentien0409/auction-bid-tracker)

A high-performance bidding engine written in Go, engineered for **ultra-low latency** and **distributed consistency**. This system demonstrates a hybrid architecture capable of running as a standalone high-speed node or a horizontally scalable distributed cluster.

---

## 🚀 Performance Comparison (Memory vs Redis)

*Measured on Apple M1 Pro. Throughput represents the absolute maximum capacity of the business logic path.*

| Scenario | Memory (RPS) | Redis (RPS) | Avg Latency (Mem/Redis) |
| :--- | :--- | :--- | :--- |
| **Hot Auction (1 Item)** | **72,000+** | **28,000+** | 2.2ms / 6.8ms |
| **Trending (10 Items)** | **69,000+** | **27,000+** | 2.4ms / 6.7ms |
| **Distributed (1000 Items)** | **72,000+** | **27,000+** | 2.2ms / 6.8ms |
| **Skewed (Zipfian)** | **68,000+** | **27,000+** | 2.4ms / 6.8ms |

> [!TIP]
> **Performance Insight**: The Memory backend provides the theoretical peak performance of the Go engine. The Redis backend, while slower due to network overhead, ensures **Atomic Distributed Consistency** across multiple nodes, making it the choice for production scale-out.

---

## 🏗 System Architecture

The project follows **Clean Architecture** principles, decoupling business domain from infrastructure details.

```text
       [ External ]         |          [ Internal / Core Logic ]
                            |
    (HTTP / JSON)           |        (Golang Domain Interfaces)
  Clients --(POST/GET)--> [ API Handler ] <---- [ Tracker Interface ]
                            |     |                   ^
                            |     |                   |
                            | [ Recovery ]     [ Bid Service ]
                            |                         |
                            |           /-------------+-------------\
                            |          v                             v
                            | [ Memory Repository ]       [ Redis Repository ]
                            | (Fine-grained Locks)        (Atomic Lua Scripts)
```

---

## 💡 Technical Design Decisions

### 1. Clean Architecture & Dependency Injection

The system follows a layered architecture:

```text
API Handler → Service Layer → Repository Interface → Concrete Implementation
```

Core business logic depends only on domain interfaces, not infrastructure details.

This allows:
- interchangeable repository implementations
- isolated unit testing
- backend-specific optimization without affecting business logic
- simplified dependency injection during runtime and tests

The active repository implementation is selected at startup:
- `MemoryRepository` for standalone low-latency execution
- `RedisRepository` for distributed coordination scenarios

---

### 2. Concurrency Strategy

The auction engine is designed for high concurrent write throughput.

#### Memory Backend
The in-memory implementation uses:
- `sync.Map` for concurrent item access
- sharded `RWMutex` locking to reduce contention
- fine-grained synchronization per auction item

This minimizes global lock contention during hot bidding scenarios.

#### Redis Backend
The distributed implementation uses:
- atomic Lua scripts executed server-side
- single-roundtrip compare-and-set updates
- Redis-native synchronization guarantees

This prevents race conditions without requiring distributed mutexes.

---

### 3. Hybrid Runtime Modes

The project intentionally separates runtime modes:

| Mode | Backend | Purpose |
| :--- | :--- | :--- |
| `make run` | Memory | Fast local development |
| `make docker-up` | Redis | Distributed environment simulation |

This separation keeps local iteration lightweight while allowing realistic distributed testing with Redis coordination.

---

### 4. Automated Verification & Performance Testing

The project includes:
- unit tests
- integration tests
- benchmarks
- synthetic load-testing scenarios

Stress tests simulate multiple traffic patterns:
- hot-item contention
- distributed low-contention traffic
- skewed Zipfian workloads

The Redis integration flow automatically provisions temporary Redis containers during integration and stress tests when needed.

---

### 5. Production-Oriented Reliability

The service includes:
- graceful shutdown handling
- panic recovery middleware
- structured logging with `slog`
- full linting and coverage validation pipeline

The goal was to build a system that remains observable and resilient under concurrent load.

---

## 🛠 Getting Started

### Prerequisites
- Go 1.24+
- Docker & Docker Compose

### Quick Start

| Mode | Backend | Command | Note |
| :--- | :--- | :--- | :--- |
| **Standalone** | In-Memory | `make run` | Fast local development without external infrastructure |
| **Standalone + Seed** | In-Memory | **`make run SEED=true`** | Pre-populated local demo mode |
| **Distributed** | Redis | `make docker-up` | Multi-container Redis-backed environment |
| **Distributed + Seed** | Redis | **`make docker-up SEED=true`** | Distributed environment with sample auction data |

> [!NOTE]
> Runtime behavior is intentionally separated:
>
> - `make run` is designed exclusively for the in-memory backend.
> - `make docker-up` is designed for the Redis-backed distributed environment.
>
> This separation keeps the standalone developer workflow lightweight while reserving Redis for integration and distributed execution scenarios.


### Full Verification Suite
The system is protected by a 100% coverage suite and automated audits.

| Command | Description |
| :--- | :--- |
| `make help` | **Gateway Command** - Display all available targets and configurations |
| **`make stress-compare`** | **Deep Performance Audit** (Memory vs Redis comparison table) |
| `make run SEED=true` | **Run & Seed** - Launch the server and automatically populate it via API |
| `make unit` | Run all unit tests for **BOTH** backends automatically |
| `make integration` | Run E2E integration tests for **BOTH** backends automatically |
| `make lint` | Run enterprise-grade static analysis |
| `make coverage` | Generate 100% coverage report |

---

## 📡 API Usage Examples

You can use the following `curl` commands to interact with the system:

### 1. Health Check
```bash
curl http://localhost:8080/health
```

### 2. Record a Bid
```bash
curl -X POST http://localhost:8080/bids \
  -H "Content-Type: application/json" \
  -d '{"item_id": "macbook-m3", "user_id": "user-1", "amount": 3000.0}'
```

### 3. Get Current Winning Bid
```bash
curl http://localhost:8080/items/macbook-m3/winning-bid
```

### 4. Get All Bids for an Item
```bash
curl http://localhost:8080/items/macbook-m3/bids
```

### 5. Get All Items a User has Bid On
```bash
curl http://localhost:8080/users/user-1/items
```

---

## 📂 Project Layout
- `internal/domain/`: Core entities and repository contracts.
- `internal/repository/`: Parallel implementations (Memory & Redis).
- `internal/service/`: High-level business orchestration.
- `tests/`: End-to-end integration scenarios.
- `.github/workflows/`: Automated CI pipeline configuration.