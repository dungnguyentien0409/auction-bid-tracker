# ⚡️ Auction Bid Tracker

[![Go Version](https://img.shields.io/badge/Go-1.22+-00ADD8?style=for-the-badge&logo=go)](https://go.dev/)
[![CI Status](https://github.com/dungnguyentien0409/auction-bid-tracker/actions/workflows/ci.yml/badge.svg)](https://github.com/dungnguyentien0409/auction-bid-tracker/actions)
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

### 1. Hybrid Storage Strategy
- **Memory**: Uses `sync.Map` and sharded `RWMutex` for zero-IO latency.
- **Redis**: Uses a **Single-Trip Lua Script** to ensure that `Compare-and-Set` logic (check if new bid > current bid) happens atomically on the database side, preventing race conditions without expensive distributed locks.

### 2. Zero-Friction Testing Matrix
Our testing suite is automated to run against **ALL** backends with zero configuration:
- **Database Isolation**: Each parallel test worker gets its own isolated Redis DB ID (0-15) to prevent cross-test data pollution.
- **Auto-Infrastructure**: The `Makefile` automatically detects, starts, and waits for Redis containers during tests if they aren't already running.

### 3. Production Readiness
- **Graceful Shutdown**: All active requests are completed before exit.
- **Panic Protection**: Middleware prevents a single bad request from crashing the server.
- **Observability**: Structured logging with `slog` for high-performance tracing.

---

## 🛠 Getting Started

### Prerequisites
- Go 1.22+
- Docker & Docker Compose

### Quick Run
```bash
# Run locally with Memory (Standalone)
make run REPO_TYPE=memory APP_ENV=development

# Run as Distributed Cluster (App + Redis)
make docker-up
```

### Full Verification Suite
The system is protected by a 100% coverage suite and automated audits.

| Command | Description |
| :--- | :--- |
| `make help` | **Gateway Command** - Display all available targets and configurations |
| **`make stress-compare`** | **Deep Performance Audit** (Memory vs Redis comparison table) |
| `make unit` | Run all unit tests for **BOTH** backends automatically |
| `make integration` | Run E2E integration tests for **BOTH** backends automatically |
| `make lint` | Run enterprise-grade static analysis |
| `make coverage` | Generate 100% coverage report |

---

## 📂 Project Layout
- `internal/domain/`: Core entities and repository contracts.
- `internal/repository/`: Parallel implementations (Memory & Redis).
- `internal/service/`: High-level business orchestration.
- `tests/`: End-to-end integration scenarios.
- `.github/workflows/`: Automated CI pipeline configuration.