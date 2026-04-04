# Legion

Distributed AI Agent Orchestration Engine built in Go — exactly-once task execution, heartbeat-based failure detection, and sub-8µs WebSocket fan-out.

**Stack:** Go · PostgreSQL · WebSockets · React · SSH/SFTP · Claude Code CLI

## What it does

Legion spins up teams of AI agents organized as a company hierarchy. A CEO agent decomposes goals into issues, delegates work down the org chart, and manages hiring, escalation, and coordination — all surfaced in an interactive React UI.

Key features:
- Interactive drag-drop org chart (React Flow) — restructure the company live
- Exactly-once task execution via Postgres advisory locks
- Agent-initiated hiring with human approval gate
- Heartbeat-based failure detection with automatic respawn
- Universal shared file system via embedded SFTP server
- Agent pause/resume with chat context injection
- Escalation bubbling up the org chart to human operator
- Pluggable runtimes: Claude Code + OpenClaw

## Quick Start (Docker)

**Prerequisites:** Docker, Docker Compose

```bash
git clone <repo>
cd conductor
docker compose up
```

Open http://localhost:3100 — register an account, create a company, and start hiring agents.

## Quick Start (Local)

**Prerequisites:** Go 1.22+, Node 20+, PostgreSQL 16

```bash
# 1. Start Postgres and run migrations
createdb conductor
psql conductor < migrations/001_initial.sql
psql conductor < migrations/002_board_agents.sql

# 2. Build the React frontend
cd web && npm install && npm run build && cd ..

# 3. Run the server
DATABASE_URL=postgres://localhost/conductor CONDUCTOR_JWT_SECRET=dev-secret go run ./cmd/conductor
```

Open http://localhost:3100.

## Remote Access via Tailscale

Access Conductor from your phone or any device without port forwarding:

```bash
# On the server
curl -fsSL https://tailscale.com/install.sh | sh
sudo tailscale up
```

Install the [Tailscale app](https://tailscale.com/download) on your phone/devices and sign into the same account. Find the server's Tailscale IP in the [admin dashboard](https://login.tailscale.com/admin/machines) and open `http://<tailscale-ip>:3100`.

The UI is mobile-responsive. React Flow org chart supports touch gestures.

## Agent Runtimes

| Runtime | Install |
|---------|---------|
| Claude Code | `npm install -g @anthropic-ai/claude-code` |
| OpenClaw | See [OpenClaw docs](https://github.com/openclaw/openclaw) |

Conductor auto-detects installed runtimes on startup. Unavailable runtimes are grayed out in the hire modal.

## Build Phases

| Phase | Scope | Status |
|-------|-------|--------|
| 1 | Core orchestrator, agent runtimes, heartbeats, Postgres state | ✅ Done |
| 2 | DAG engine, escalation bubbling FSM, advisory locks | ✅ Done |
| 3 | Embedded SFTP server, path-based ACL, permission inheritance | ✅ Done |
| 4 | React UI: dashboard, org chart, issues, hiring, audit log | ✅ Done |
| 5 | Agent chat, FS browser, cost tracking UI | ✅ Done |
| 6 | Docker deployment, Tailscale docs, multi-company isolation | ✅ Done |

## Testing

```bash
# Go backend (unit + integration tests, uses in-memory mocks — no DB required)
go test ./...

# React frontend (Vitest)
cd web && npx vitest run
```

Coverage: 718 frontend tests across 27 files; full backend coverage for orchestrator, store, API handlers, heartbeat, agent runtimes, and WebSocket hub.

## Benchmarks

Measured on Apple M5, Go 1.22, `go test -bench=. -benchmem`. Run with `go test ./internal/... -bench=.`.

### WebSocket fan-out (`internal/ws`)

| Scenario | Latency | Allocs |
|----------|---------|--------|
| Broadcast → 1 client | 275 ns | 6 |
| Broadcast → 10 clients | 1.36 µs | 6 |
| Broadcast → 100 clients | 7.6 µs | 6 |
| 10 concurrent writers → 50 clients | 444 ns | 6 |
| BroadcastAll (5 companies × 10 clients) | 4.25 µs | 3 |
| Register + unregister | 110 ns | 4 |

Fan-out scales linearly at ~76 ns per additional client. Zero additional allocations regardless of subscriber count — the event is marshaled once and the same bytes are enqueued to every channel.

### Stdout parser (`internal/stdout`)

| Operation | Latency | Allocs |
|-----------|---------|--------|
| `IsControlLine` — hit | 1.9 ns | 0 |
| `IsControlLine` — miss | 7.4 ns | 0 |
| `ParseLine` (HEARTBEAT) | 69 ns | 3 |
| `ParseLine` (HIRE with JSON) | 170 ns | 3 |
| `DecodeHire` (full JSON unmarshal) | 453 ns | 7 |

Every line of agent stdout passes through `IsControlLine` at ~2 ns with zero allocations.

### Heartbeat watcher (`internal/heartbeat`)

| Operation | Latency | Allocs |
|-----------|---------|--------|
| Check cycle — no stale agents | 1.6 ns | 0 |
| Check cycle — stale agent found | 1.85 µs | 7 |
| `RecordHeartbeat` | 0.64 ns | 0 |
| `RecordHeartbeat` (parallel) | 0.19 ns | 0 |
| Failure detection → callback | 2.3 µs | 20 |
| Watch + Unwatch 50 agents | 47 µs | 459 |

Detection callback fires in ~2.3 µs from the check cycle; actual wall-clock detection window is bounded by `CheckInterval` (20s) + `StaleThreshold` (20s).

### Orchestrator coordination (`internal/orchestrator`)

| Operation | Latency | Allocs |
|-----------|---------|--------|
| Issue checkout — sequential | 1.6 ns | 0 |
| Issue checkout — 10 concurrent workers | 25 ns | 0 |
| Issue checkout — 50 concurrent workers | 26 ns | 0 |
| Wake-channel dispatch (assign loop signal) | 130 ns | 0 |
| RWMutex read (runtime map, concurrent) | 40 ns | 0 |
| RWMutex 90% read / 10% write | 13 ns | 0 |

Checkout contention is near-flat from 10→50 concurrent workers (25 ns → 26 ns), confirming the coordinator doesn't bottleneck under agent-scale concurrency.

## Architecture

See the inline doc comments in:
- `internal/orchestrator/orchestrator.go` — central coordinator
- `internal/agent/runtime.go` — pluggable AgentRuntime interface
- `internal/sftp/server.go` — embedded SFTP server
- `internal/ws/hub.go` — WebSocket fan-out gateway
