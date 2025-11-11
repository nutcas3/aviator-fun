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

## Extending the Backend: Supporting Other Crash Game Types

The core Aviator backend architecture is robust and highly scalable. To support variations like JetX, Mines, and Plinko (as seen on platforms like Odibets), you primarily need to introduce new Game Manager modules and specific API logic.

### 1. Core Concept: The Game Engine Interface

The key to supporting multiple game types is defining a common interface for the game engine.

| Game Module | Core Responsibility |
| --- | --- |
| **AviatorGame** | Manages the rising multiplier and the crash point using the Provably Fair system. |
| **MinesGame** | Manages the grid state, handles tile clicks, and determines the location of the mines. |
| **PlinkoGame** | Determines the ball's landing slot and its resulting multiplier using the Provably Fair system. |

### 2. Building the Variations (What to Modify)

#### A. Classic Curve Variations (e.g., JetX, Spaceman)

- **Modification**: Minimal. The existing `AviatorGame` logic handles the rising curve (`round_running` and `crash` events).
- **Backend Task**: Add a `GameType` field to the database/Redis model (e.g., `type: 'aviator'`, `type: 'jetx'`). The core logic remains the same; the front-end handles the visual change (a jet instead of a plane).

#### B. Mines Game Implementation

- **Game Logic**: Needs a new `MinesGame` manager.
- **Setup**: Generate a 5×5 grid and secretly place a random number of "Mines" (e.g., 3, 5, or 7) using the Provably Fair seed to determine their positions.
- **Betting**: The `POST /api/v1/game/bet` endpoint must include the number of mines the player is playing with.
- **Action**: Add a new API endpoint: `POST /api/v1/game/click_tile { bet_id: "...", tile_id: 12 }`. The backend verifies the click:
  - **If Safe**: Update the player's potential payout.
  - **If Mine**: Instantly crash the game for that user, and log the loss.
- **Cashout**: Add a cashout endpoint, `POST /api/v1/game/cashout`, which finalizes the bet at the current potential payout.

#### C. Plinko Game Implementation

- **Game Logic**: Needs a new `PlinkoGame` manager.
- **Action**: Add a new API endpoint: `POST /api/v1/game/drop_ball { bet_id: "...", row_count: 8 }`. The `row_count` determines the risk.
- **Result**: Use the Provably Fair seed to determine the final horizontal landing position (e.g., which slot the ball lands in) and the final multiplier instantly. There are no ongoing updates—the result is calculated immediately upon the "drop."

### 📋 Extended API Overview (For New Game Types)

The base API must be extended to support the different interaction models:

#### 🚀 Mines Game Endpoints (Tile-Click Model)

| Endpoint | Description | Interaction Type |
| --- | --- | --- |
| `POST /api/v1/mines/bet` | Place a bet and set the number of mines. | REST |
| `POST /api/v1/mines/click` | Reveal a tile (Win/Mine result). | REST |
| `POST /api/v1/mines/cashout` | Cash out the current accumulated win. | REST |

#### 🎯 Plinko Game Endpoints (Instant Result Model)

| Endpoint | Description | Interaction Type |
| --- | --- | --- |
| `POST /api/v1/plinko/drop` | Place a bet and initiate the ball drop. Returns the final multiplier. | REST |

### 🔑 Provably Fair System Variations

The core principle remains HMAC-SHA256, but the seed result is interpreted differently for each game:

| Game Type | Seed Interpretation |
| --- | --- |
| **Aviator** | The HMAC-SHA256 result (a hex string) is converted into a floating-point number representing the crash multiplier. |
| **Mines** | The HMAC-SHA256 result is used to generate a sequence of pseudo-random numbers, which are mapped to the Mine grid coordinates. |
| **Plinko** | The HMAC-SHA256 result is used to generate a sequence of random left/right movements for the ball as it falls through the pegs, determining the final landing slot. |

### ✅ Next Steps for Development

Your existing Makefile and Docker Compose setup are perfect for starting this expansion.

1. **Refactor**: Create a `game_manager` interface in Go that `AviatorGame`, `MinesGame`, and `PlinkoGame` all implement.
2. **New Routes**: Implement the new REST endpoints for Mines and Plinko (as listed above).
3. **New Logic**: Build the `MinesGame` and `PlinkoGame` logic, focusing heavily on correctly translating the Provably Fair seed into the game outcome.

**Would you like a detailed walkthrough of how to derive the Mines grid coordinates from the HMAC-SHA256 seed?**

---

## 🎮 Complete Game Ecosystem Architecture

### Game Categories and Providers

The platform supports multiple game categories, each with different mechanics and providers:

#### 1. Crash Game Variations (Instant Games)

| Category | Provider | Mechanic | Description |
|----------|----------|----------|-------------|
| **Mines/Tile Games** | Spribe, Turbo Games | Tile-Click | Players click tiles to reveal multipliers while avoiding hidden mines |
| **Plinko** | Spribe, Turbo Games | Peg-Board | Ball drops through pegs to land in multiplier slots |
| **Dice/HiLo** | Spribe, Turbo Games | Probability | Bet on number ranges with adjusted payouts |
| **Virtuals** | Elbet, Golden Race | Simulated Sports | Fast-paced graphical sports simulations |

#### 2. Casino Slots (315+ Games)

| Category | Provider | Examples | Key Feature |
|----------|----------|----------|-------------|
| Video Slots | Pragmatic Play, Booming | Gates of Olympus, Sweet Bonanza | Spinning reels, bonus rounds |
| Drops and Wins | Pragmatic Play | Network Promotions | Linked progressive jackpots |

#### 3. Live Casino (169+ Tables)

| Category | Provider | Examples | Key Feature |
|----------|----------|----------|-------------|
| Live Games | Evolution, Pragmatic | Roulette, Blackjack | Real dealers, live streaming |
| Game Shows | Evolution | Crazy Time, Dream Catcher | Interactive entertainment games |

### 🏗️ Modular Backend Architecture

The existing Go, Fiber, Redis, and PostgreSQL stack can be extended with a modular game engine system:

```
┌────────────────────────────────┐
│      Go Backend (API/WS)       │
├────────────────────────────────┤
│    Game Factory / Router       │  <-- Routes requests by GameType
├────────────────┬───────────────┤
│ ┌──────────────▼──────────┐ ┌──────────────▼──────────┐ ┌──────────────▼──────────┐
│ │ 🚀 Aviator Engine (WS)  │ │ ⛏️ Mines Engine (REST)  │ │ 🎲 Plinko/Dice (REST)   │
│ ├─────────────────────────┤ ├─────────────────────────┤ ├─────────────────────────┤
│ │ - Real-Time Multiplier  │ │ - Grid State Management │ │ - Instant Seed Mapping  │
│ │ - Per-Round Broadcast   │ │ - Tile Click Logic      │ │ - High-Roll Calculation │
│ └─────────────────────────┘ └─────────────────────────┘ └─────────────────────────┘
```

### Backend Implementation Focus for Non-Curve Games

| Game Type | Key Backend Changes in Go | Provably Fair Logic Change |
|-----------|---------------------------|----------------------------|
| **Mines** | New REST Endpoints: `POST /mines/bet`, `POST /mines/click`, `POST /mines/cashout`. Uses Redis to store the current unrevealed grid state per user. | The Provably Fair seed determines the exact position of all mines at the start of the round. The client verifies the full mine layout. |
| **Plinko/Dice** | Single REST Endpoint: `POST /plinko/drop` or `POST /dice/roll`. This must be a single, transactional operation. | The Provably Fair seed is used to instantly map to the final landing slot (Plinko) or the final roll number (Dice). No real-time updates are needed. |
| **Slots** | Integration: Slots are typically provided by the vendors (e.g., Pragmatic Play). You build a seamless API Wrapper that authenticates the player and forwards the spin request to the vendor's external API, then handles the balance update. | The random number generation (RNG) is handled entirely by the vendor's certified server (no custom Provably Fair needed). |

### Core Infrastructure Foundation

The existing system's use of **Redis for fast state management** and **PostgreSQL for immutable transaction logging** is essential and remains the foundation for all these diverse game types.

### Integration Strategy

1. **Core Services** (Existing):
   - User authentication/authorization
   - Wallet/balance management
   - Transaction logging
   - WebSocket infrastructure

2. **Game Engine Layer** (New):
   - `GameEngine` interface for all game types
   - Common game lifecycle management
   - Standardized event system

3. **Game Modules** (Modular):
   - Each game type implements the `GameEngine` interface
   - Self-contained game logic and state management
   - Game-specific API endpoints

4. **Vendor Integration** (For Slots/Live):
   - Unified API gateway for external providers
   - Session management
   - Bet resolution handling

### Getting Started with a New Game Type

1. **Define the Game Contract**:
   - Game parameters and rules
   - Betting model
   - Payout structure

2. **Implement Core Logic**:
   - Game state management
   - Win/loss determination
   - Integration with provably fair system

3. **Create API Endpoints**:
   - Game-specific endpoints
   - WebSocket events (if real-time)
   - Admin controls

4. **Integrate with Core Services**:
   - User authentication
   - Balance updates
   - Transaction logging

5. **Add Monitoring**:
   - Game-specific metrics
   - Error tracking
   - Performance monitoring

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
