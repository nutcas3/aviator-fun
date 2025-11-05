# 🎮 Crash Game Backend (Aviator)

High-performance, real-time crash game backend built with Go, Fiber, Redis, and PostgreSQL. The system is designed for thousands of concurrent players, provably fair gameplay, and production-grade reliability.

---

## Quick Start

### Prerequisites

- Go 1.21+
- Redis 7+
- PostgreSQL 14+
- Docker & Docker Compose (optional but recommended)

### Instructions

1.  **Clone and configure**

    ```bash
    git clone https://github.com/nutcas3/aviator-fun.git
    cd aviator
    cp .env.example .env
    # Edit .env with your configuration if needed (defaults are fine for local)
    ```

2.  **Run with Docker (Recommended)**

    ```bash
    # This will start the Go app, PostgreSQL, and Redis
    make docker-run
    ```

3.  **Run Locally**

    If you prefer not to use Docker for the Go app, you can run it directly. You'll still need PostgreSQL and Redis running.

    ```bash
    # 1. Start database and cache (if not already running)
    # You can use Docker for this:
    # docker run -d --name postgres ...
    # docker run -d --name redis ...

    # 2. Run the application
    make run
    ```

4.  **Check Health**

    Once running, check the health endpoint:
    ```bash
    curl http://localhost:3000/health
    ```

---

## Makefile Commands

| Command                     | Description                                          |
| --------------------------- | ---------------------------------------------------- |
| `make all`                  | Build and test the project                           |
| `make build`                | Compile the API binary                               |
| `make run`                  | Run the API directly (`go run cmd/api/main.go`)      |
| `make docker-run`           | Start the full stack (app, db, cache) via Docker     |
| `make docker-down`          | Stop all Docker containers                           |
| `make watch`                | Live reload using `air` (auto-installs if missing)   |
| `make test`                 | Run unit tests (skips integration tests)             |
| `make test-all`             | Run the full test suite, including integration tests |
| `make itest`                | Run database integration tests only                  |
| `make migrate-up`           | Apply all pending database migrations                |
| `make migrate-down`         | Roll back the last database migration                |
| `make migrate-version`      | Show the current migration version                   |
| `make migrate-create name=<name>` | Scaffold a new migration file                        |
| `make db-reset`             | Convenience: `down` then `up`                        |
| `make clean`                | Remove build artifacts                               |

---

## Architecture Overview

```
┌─────────────────────────────────────────────────────────────┐
│                      Load Balancer                          │
│                    (Nginx/HAProxy)                          │
└────────────────────┬────────────────────────────────────────┘
                     │
        ┌────────────┴────────────┬────────────┐
        │                         │            │
┌───────▼────────┐    ┌──────────▼─────┐  ┌──▼──────────┐
│  Go Backend 1  │    │  Go Backend 2  │  │  Go Backend N│
│  (Fiber+WS)    │    │  (Fiber+WS)    │  │  (Fiber+WS)  │
└───────┬────────┘    └──────────┬─────┘  └──┬───────────┘
        │                        │            │
        └────────────┬───────────┴────────────┘
                     │
         ┌───────────┴────────────┐
         │                        │
    ┌────▼─────┐          ┌──────▼──────┐
    │  Redis   │          │ PostgreSQL  │
    │ Cluster  │          │   Primary   │
    │(Sentinel)│          │ + Replicas  │
    └──────────┘          └─────────────┘
```

### Core Features
- Real-time WebSocket engine with per-round broadcasts
- Provably fair algorithm using HMAC-SHA256
- Atomic bet processing backed by Redis
- Auto-cashout and manual cashout support
- High-concurrency safe game manager (goroutines + channels)

### Production Enhancements
- Graceful shutdown & panic recovery middleware
- Structured logging and health endpoints
- Database migrations via `golang-migrate`
- Docker + Compose orchestration
- Rate limiting, CORS, connection pooling

---

## API Overview

### REST Endpoints

- `GET /health` – Database, cache, and game status
- `GET /api/v1/game/state` – Current round state
- `POST /api/v1/game/bet` – Place a bet
- `POST /api/v1/game/cashout` – Cash out a bet
- `GET /api/v1/user/:userId/balance` – Fetch user balance
- `POST /api/v1/user/:userId/balance` – Update balance (admin/testing)

### WebSocket

Connect: `ws://localhost:3000/ws?user_id=<id>`

**Client → Server**
- `place_bet` – `{ "type": "place_bet", "amount": 100, "auto_cashout": 2.5 }`
- `cashout` – `{ "type": "cashout", "bet_id": "BET-..." }`
- `ping`

**Server → Client**
- `initial_state`, `round_start`, `round_running`
- `update` (multiplier tick), `crash`
- `bet_placed`, `cashout`

---

## Provably Fair System

1. Server generates a secret `server_seed` and publishes `hash_commitment = SHA256(server_seed)`.
2. Players place bets while only the commitment is known.
3. After the crash, the server reveals `server_seed`. Players verify with:

```
HMAC-SHA256(server_seed, client_seed:nonce) → crash_multiplier
```

Use `POST /api/v1/game/verify` to validate multipliers client-side.

---

## Testing

This project has comprehensive test coverage for all major components.

### Running Tests

```bash
# Run unit tests (fast, skips integration tests)
make test

# Run the full test suite (requires Docker to be running)
make test-all

# Run only database integration tests
make itest

# Generate a coverage report
go test ./... -coverprofile=coverage.out && go tool cover -html=coverage.out
```

### Test Categories
- **Unit Tests**: Provably fair logic, WebSocket hub, data structures, and handlers.
- **Integration Tests**: Database operations using `testcontainers` (skipped by default).
- **Concurrency Tests**: Thread-safe operations for broadcasting and client management.
- **Performance Benchmarks**: Key algorithms are benchmarked. Run with `go test ./internal/game -bench=. -benchmem`.

---

## Security & Production

- **Checklist**: Enable TLS, implement JWT auth, configure Redis auth, set CORS policies, and monitor system metrics.
- **Scaling**: The architecture supports horizontal scaling of the Go backend instances, Redis (via Sentinel/Cluster), and PostgreSQL (via read replicas).

---

## Troubleshooting

| Issue | Resolution |
| --- | --- |
| Redis connection refused | Ensure Redis is running and `REDIS_URL` in `.env` is correct. |
| Migrations fail | Run `make migrate-up` and check the database connection string. |
| Integration tests hang | Run `make test` to skip them, or ensure Docker is running before `make test-all`. |

---

## Contributing

1. Fork the repository
2. Create a feature branch
3. Submit a Pull Request with tests and documentation updates

---

**Just a simple crash game backend by Nutcase**
