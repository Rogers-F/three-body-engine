# Three-Body Engine

Automated 7-phase workflow engine that orchestrates cross-review between multiple code agents (Claude, Codex, Gemini). The "Three-Body" model ensures no agent reviews its own output — every artifact is challenged by an independent reviewer before proceeding.

## Why

LLMs are bad at self-review. They tend to approve their own work and skip steps when given the option. This engine solves both problems:

- **Mandatory 7 phases** — the Go FSM enforces the full pipeline. No shortcuts, no downgrades.
- **Cross-review** — the producer is never the reviewer. Weighted voting (Codex 0.45 / Claude 0.30 / Gemini 0.25) aggregates independent opinions.
- **Environment-level enforcement** — hooks and guards run in the Go engine, not in the LLM's context. The LLM cannot bypass them.

## Architecture

```
                        ┌───────────────────────────┐
                        │   Desktop (React + Vite)  │
                        │   Workflow Canvas / Views  │
                        └─────────────┬─────────────┘
                                      │ HTTP / SSE
                        ┌─────────────▼─────────────┐
                        │   IPC Layer (Go HTTP)      │
                        ├───────────────────────────┤
                        │   Bridge Layer             │
                        │   Session Manager          │
                        ├───────────────────────────┤
                        │   Guard Layer              │
                        │   Budget / Permission /    │
                        │   Rate Limit / Round Cap   │
                        ├───────────────────────────┤
                        │   Workflow Engine (FSM)    │
                        │   7-Phase State Machine    │
                        │   Gate Registry / Cost Gov │
                        ├───────────────────────────┤
                        │   Team Layer               │
                        │   Worker Lifecycle /       │
                        │   Supervisor / Permissions │
                        ├───────────────────────────┤
                        │   Review Engine            │
                        │   ScoreCard / Consensus /  │
                        │   Blocker Detection        │
                        ├───────────────────────────┤
                        │   Store (SQLite WAL)       │
                        │   Tasks / Events / Intents │
                        │   Workers / ScoreCards     │
                        └───────────────────────────┘
```

## 7-Phase Workflow

```
A ──► B ──► C ──► D ──► E ──► F ──► G
                  ▲     │           │
                  └─────┘           │     D→C: rollback (redesign)
                        ▲           │     F→E: rework (fix issues)
                        └───────────┘
```

| Phase | Name | Mode |
|-------|------|------|
| A | Goal Clarification | Human-in-the-loop (only phase requiring user) |
| B | Exploration | Auto, parallel Workers possible |
| C | Design | Auto, Lead only |
| D | Three-Body Review | Auto, Codex + Gemini cross-review |
| E | Implementation | Auto, parallel Workers possible |
| F | Cross-Acceptance | Auto, Codex + Gemini acceptance |
| G | Test & Deliver | Auto, parallel Testers possible |

After Phase A, the user does not participate. The engine runs autonomously until delivery or budget exhaustion.

## Project Structure

```
three-body-engine/
├── engine/                        # Go backend (8,800+ LOC)
│   ├── cmd/threebody/main.go      # Entry point, wires all layers
│   └── internal/
│       ├── domain/                # Core types + error codes
│       ├── store/                 # SQLite repos (7 repositories)
│       ├── workflow/              # FSM, gates, budget governor
│       ├── team/                  # Worker lifecycle, supervisor, permissions
│       ├── review/                # ScoreCard schema, consensus, blockers
│       ├── guard/                 # Budget + permission + rate limit checks
│       ├── mcp/                   # Provider registry, session management
│       ├── bridge/                # Provider-agnostic session orchestration
│       ├── config/                # JSON config loader with validation
│       └── ipc/                   # HTTP API handlers + SSE streaming
│
├── desktop/                       # React frontend (2,600+ LOC)
│   └── src/
│       ├── api/                   # HTTP client, SSE wrapper, React hooks
│       ├── components/
│       │   ├── common/            # Button, Card, Badge, ProgressBar
│       │   ├── layout/            # Shell, Sidebar, TopBar
│       │   ├── workflow/          # Canvas, PhaseNode, WorkerNode, edges
│       │   ├── review/            # ScorecardGrid, ConsensusBar, IssueList
│       │   └── cost/              # BudgetGauge, TokenChart, PhaseCostBreakdown
│       ├── views/                 # Dashboard, Review, Cost, Settings
│       ├── stores/                # Zustand (workflow, canvas, UI)
│       ├── design-system/         # Tokens, ThemeProvider (light/dark)
│       └── types/                 # TypeScript domain types
│
└── .github/workflows/             # CI: test + build + cross-compile release
```

## Getting Started

### Prerequisites

- Go 1.22+
- Node.js 22+
- npm

### Build

```bash
# Backend
cd engine
go build -o threebody ./cmd/threebody

# Frontend
cd desktop
npm install
npm run build
```

### Run

Create a config file (`config.json`):

```json
{
  "db_path": "./threebody.db",
  "workspace": ".",
  "budget_cap_usd": 10.0,
  "listen_addr": ":9800",
  "providers": {
    "codex": {
      "command": "codex",
      "args": ["--mcp"]
    },
    "gemini": {
      "command": "gemini",
      "args": ["--mcp"]
    }
  }
}
```

```bash
./threebody --config config.json
```

The engine starts on `http://localhost:9800`. The frontend connects automatically (configure via `VITE_API_URL` env var or the Settings view).

### Test

```bash
# Go tests (173 tests, includes race detection)
cd engine && go test -race ./...

# Frontend type check
cd desktop && npx tsc --noEmit

# Frontend build
cd desktop && npm run build
```

## API Endpoints

| Method | Path | Description |
|--------|------|-------------|
| `GET` | `/healthz` | Health check |
| `POST` | `/api/v1/flow` | Create a new workflow |
| `GET` | `/api/v1/flow/{taskID}` | Get workflow state |
| `POST` | `/api/v1/flow/{taskID}/advance` | Advance to next phase |
| `GET` | `/api/v1/flow/{taskID}/events` | List workflow events |
| `GET` | `/api/v1/flow/{taskID}/events/stream` | SSE event stream |
| `GET` | `/api/v1/flow/{taskID}/workers` | List workers |
| `GET` | `/api/v1/flow/{taskID}/reviews` | List review scorecards |
| `GET` | `/api/v1/flow/{taskID}/cost` | Get cost summary |

### Example

```bash
# Create a flow
curl -X POST http://localhost:9800/api/v1/flow \
  -H 'Content-Type: application/json' \
  -d '{"task_id": "task-001", "budget_cap_usd": 10.0}'

# Advance through phases
curl -X POST http://localhost:9800/api/v1/flow/task-001/advance \
  -H 'Content-Type: application/json' \
  -d '{"action": "advance", "actor": "lead"}'

# Stream events (SSE)
curl -N http://localhost:9800/api/v1/flow/task-001/events/stream
```

## Key Design Decisions

| Decision | Rationale |
|----------|-----------|
| Mandatory 7 phases, no risk-based routing | LLMs will always choose the shortcut if given one |
| Go engine enforces all guards | Shell hooks are thin wrappers; logic lives in Go for portability |
| SQLite WAL with MaxOpenConns(1) | Single-writer guarantees consistency; WAL allows concurrent reads |
| Lead persistent + Workers ephemeral | Avoids context bloat from long-lived workers; ContextDigest carries state across spawns |
| Three-way merge for conflicts, fail-to-user on failure | MVP does not trust LLM conflict resolution |
| Compaction with 9 mandatory slots | Prevents Lead context from exceeding 200k tokens across phases |
| Intent Log with idempotency keys | Ensures Worker kill+respawn doesn't duplicate file operations |

## Configuration

| Field | Default | Description |
|-------|---------|-------------|
| `db_path` | (required) | Path to SQLite database |
| `workspace` | (required) | Project workspace root |
| `budget_cap_usd` | (required) | Maximum cost per task in USD |
| `listen_addr` | `:9800` | HTTP server listen address |
| `check_interval_sec` | `10` | Supervisor heartbeat check interval |
| `heartbeat_max_age` | `60` | Max seconds before worker is considered unresponsive |
| `max_concurrent_workers` | `5` | Maximum workers per task |
| `max_rounds` | `3` | Maximum rollback/rework cycles |
| `rate_limit_per_minute` | `60` | Per-task API rate limit |
| `providers` | `{}` | Map of provider name to command config |

## CI / Release

Push a `v*` tag to trigger the release pipeline:

```bash
git tag -a v0.1.0 -m "v0.1.0"
git push origin v0.1.0
```

The GitHub Actions workflow will:
1. Run Go tests (`-race`) + TypeScript type check
2. Build the frontend
3. Cross-compile Go binary for Windows/macOS/Linux (amd64 + arm64)
4. Package archives with frontend dist bundled
5. Create a GitHub Release with checksums

## Stats

- **98 source files** (32 Go + 25 Go test + 41 TypeScript)
- **~11,400 LOC** (8,800 Go + 2,600 TypeScript)
- **173 tests** with race detection
- **4 platform targets** (Windows, macOS amd64/arm64, Linux)

## License

MIT
