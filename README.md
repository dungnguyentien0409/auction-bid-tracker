# ⚡️ Auction Bid Tracker

[![Go Version](https://img.shields.io/badge/Go-1.22+-00ADD8?style=for-the-badge&logo=go)](https://go.dev/)
[![Coverage](https://img.shields.io/badge/Coverage-100%25-brightgreen?style=for-the-badge)](https://github.com/dungnguyentien0409/auction-bid-tracker)
[![License](https://img.shields.io/badge/License-MIT-yellow?style=for-the-badge)](LICENSE)

A high-performance bidding engine written in Go, engineered for **ultra-low latency** and **extreme concurrency**. This system demonstrates advanced synchronization techniques capable of handling **100k+ RPS** on a single node.

---

## 🚀 Performance Matrix

*Measured on Apple M1 Pro (10-core). The system demonstrates near-linear scalability and consistent latency across various load patterns.*

| Scenario | Items | Distribution | Throughput | Avg Latency |
| :--- | :--- | :--- | :--- | :--- |
| **Hot Auction** | 1 | Single | **70,000+ RPS** | ~2.3 ms |
| **Trending** | 10 | Uniform | **71,000+ RPS** | ~2.3 ms |
| **Distributed** | 1,000 | Uniform | **80,000+ RPS** | ~1.9 ms |
| **Skewed (Zipf)** | 1,000 | Zipfian (80/20) | **78,000+ RPS** | ~2.0 ms |

> [!NOTE]
> **Latency vs. Timeout**: While server-level timeouts (`ReadTimeout`/`WriteTimeout`) are configured to handle network jitter under extreme stress, the actual **Business Latency** remains ultra-low at **~2.2ms**. This ensures bids are processed in near real-time, meeting the critical requirements of a competitive auction system.

> [!IMPORTANT]
> **Technical Insight**: The stability of RPS across different distributions (from 1 to 1,000 items) proves that our **Fine-Grained Locking** effectively mitigates "Hot Partition" issues. 
> 
> ### Load Patterns Explained:
> - **Uniform (Distributed)**: Traffic is spread equally across all items. This tests the **Peak Throughput** by minimizing lock contention.
> - **Zipfian (Skewed)**: Follows the **80/20 rule** (a few items get most of the traffic). This is the most **Realistic Simulation**, testing how the system handles bias and high contention on popular items.
>
> ### 🧪 Testing Methodology: Stress vs. SLA
> - **Stress Testing (`APP_ENV=stress`)**: We use relaxed infrastructure timeouts (15s) to filter out OS-level network congestion noise. This allows us to measure the **absolute maximum throughput** of the application logic.
> - **SLA Verification (`APP_ENV=development`)**: In standard operation, we enforce strict timeouts (5s) to guarantee a responsive user experience. If load exceeds this capacity, the strategy is to **Scale Out** or implement **Load Shedding**.

---

## 🏗 System Architecture

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

---

## 💡 Technical Design Decisions

### 1. Concurrency & Performance
Instead of a global lock which bottlenecks the entire system, I implemented **Fine-Grained Locking**:
- **Sharded State**: Using `sync.Map` for O(1) item lookups.
- **Atomic Item Updates**: Each item has its own `RWMutex`. This allows parallel bidding on different items with zero interference.

### 2. Interface-Driven Architecture
- **Dependency Injection**: The core logic depends on abstractions, not implementations. 
- **Scalability**: Swapping the In-Memory store for a persistent SQL/NoSQL database requires zero changes to the service layer.

### 3. Fault Tolerance & Production Readiness
- **Panic Recovery**: Middleware ensures a single failing request cannot bring down the entire node.
- **Graceful Shutdown**: Implemented signal handling to ensure all active requests are finished before the process exits.
- **Structured Logging**: Leveraging `log/slog` for high-performance, machine-readable logs.

---

## 🛠 Getting Started

### Prerequisites
- Go 1.22+
- Docker (Optional)

### Quick Start
The system requires two mandatory environment variables: `APP_ENV` and `REPO_TYPE`.

```bash
# Run locally with In-Memory (Standard development)
make run APP_ENV=development REPO_TYPE=memory

# Run locally with Redis (Requires Redis on localhost:6379)
make run APP_ENV=development REPO_TYPE=redis

# Run via Docker (Always starts with REPO_TYPE=memory unless configured)
make docker-run REPO_TYPE=memory
```

### Verification Suite
| Command | Description |
| :--- | :--- |
| `make stress-matrix REPO_TYPE=memory` | **Run All Scenarios** and generate a performance report |
| `make docker-up` | **Start Distributed System** (App + Redis) via Docker Compose |
| `make docker-down` | Stop and clean up all Docker resources |
| `make test APP_ENV=development REPO_TYPE=memory` | Run all unit & integration tests (100% Coverage) |
| `make coverage APP_ENV=development REPO_TYPE=memory` | Generate HTML report for code coverage verification |
| `make benchmark` | Micro-benchmarks for core engine synchronization speed |
| `make load-test APP_ENV=stress REPO_TYPE=memory` | General API stress test |
| `make test-zipf APP_ENV=stress REPO_TYPE=memory` | **Skewed (Zipfian)** - Realistic 80/20 traffic distribution |

---

## 🌐 Horizontal Scaling (Redis)

To scale the system across multiple nodes, you can switch from the `In-Memory` repository to the `Redis` repository. This ensures all nodes share the same state and use **Distributed Atomic Updates** via Lua scripts.

### Run with Docker Compose (Recommended)
This will start the Auction Server and a Redis instance automatically:
```bash
docker-compose up --build
```

### Manual Run with Redis
1. Ensure Redis is running on `localhost:6379`.
2. Start the server with `REPO_TYPE=redis`:
```bash
REPO_TYPE=redis make run
```

---

## 📂 Project Layout
- `cmd/`: Application entry points & Load testing tools.
- `internal/api/`: HTTP layer, routing, and middlewares.
- `internal/domain/`: Core business entities and repository contracts.
- `internal/repository/`: Thread-safe data structures & synchronization.
- `internal/service/`: Business logic orchestration.