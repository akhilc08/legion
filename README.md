# Conductor

Distributed AI Agent Orchestration Engine. A Go-based rebuild of [Paperclip](https://github.com/paperclipai/paperclip) with real distributed systems guarantees.

**Stack:** Go · PostgreSQL · WebSockets · React · SSH/SFTP · Claude Code CLI · OpenClaw

## What it does

Conductor spins up teams of AI agents (Claude Code or OpenClaw) organized as a company hierarchy. A CEO agent decomposes goals into issues, delegates work down the org chart, and manages hiring, escalation, and coordination — all surfaced in an interactive React UI.

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

## Architecture

See the inline doc comments in:
- `internal/orchestrator/orchestrator.go` — central coordinator
- `internal/agent/runtime.go` — pluggable AgentRuntime interface
- `internal/sftp/server.go` — embedded SFTP server
- `internal/ws/hub.go` — WebSocket fan-out gateway
