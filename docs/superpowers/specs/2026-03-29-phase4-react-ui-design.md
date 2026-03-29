# Phase 4: React UI Design

**Date:** 2026-03-29
**Status:** Approved

---

## Overview

Build the React frontend for the legion conductor platform. The UI connects to the existing Go backend via REST and WebSocket and provides operators with a live view of their AI agent companies.

---

## Stack

| Concern | Choice |
|---------|--------|
| Bundler | Vite + React + TypeScript |
| Styling | Tailwind CSS v3 |
| Components | shadcn/ui (zinc/slate dark theme) |
| Server state | TanStack React Query v5 |
| UI state | Zustand |
| Routing | React Router v6 |
| Org chart | React Flow v12 |
| HTTP client | Axios (instance with JWT interceptor) |

---

## Layout

Sidebar navigation app. Fixed left sidebar contains the company selector (dropdown at bottom), nav links, and notification badges. The main area renders the active page.

Sidebar links:
- Dashboard
- Org Chart
- Issues
- Hiring (badge: count of pending hires)
- Audit Log

---

## Pages

### Dashboard

Status summary at a glance:
- Agent status counts (idle / working / blocked / failed / degraded) as stat cards
- Active escalations list with trigger reason and current assignee
- Pending hire requests with quick approve/reject
- Recent notifications (dismissible)

All data refreshes via React Query + WebSocket patches.

### Org Chart

React Flow canvas. Each node represents an agent:
- Node color encodes status (green=working, yellow=idle, red=blocked/failed)
- Node label: role title + status dot
- Edges represent manager→report relationships (directed, top-down layout)

Interactions:
- **Drag node** → reassign manager (`POST /agents/{id}/reassign`). On drop, confirm with a small toast before committing.
- **Click node** → opens a slide-in detail panel on the right side of the canvas. Panel contains: status badge, current issue summary, token spend vs budget progress bar, quick actions (Spawn, Kill, Pause/Resume, Open Chat).
- **Toolbar**: zoom fit, layout refresh button.

### Issues

Table view with columns: title, assignee, status, created_at, attempt count.
Filters: status (multi-select), assignee (dropdown), search by title.
Click a row to expand an inline detail with: description, dependency chain (DAG), failure reason if failed, escalation chain if escalated.

### Hiring

Card list of `PendingHire` records showing: requested role, requesting agent, reporting-to agent, runtime, budget allocation, initial task.
Each card has Approve and Reject buttons (`POST /hires/{id}/approve` or `/reject`).

### Audit Log

Scrollable table: timestamp, event type, actor, payload (collapsible JSON).
Infinite scroll or pagination (50 per page).

---

## Real-Time (WebSocket)

One WebSocket connection per active company, opened at `GET /api/companies/{id}/ws?token=<jwt>`. Browsers cannot send custom headers on WebSocket connections, so the JWT is passed as a query parameter. The Go WS handler reads `r.URL.Query().Get("token")` and validates it instead of using the `Authorization` header middleware.

The backend pushes JSON messages for:
- Agent status change
- Issue status change
- New escalation / escalation resolved
- New notification
- Hire approved / rejected

On each message the Zustand store is patched in-place. React Query cache is also invalidated for the relevant query key so any stale refetch picks up fresh data.

---

## Auth

- `POST /api/auth/login` → JWT stored in `localStorage`
- Axios instance attaches `Authorization: Bearer <token>` to every request
- On 401 response, clear token and redirect to `/login`
- `/login` and `/register` are public routes; everything else requires auth
- After login, redirect to `/` (company list) or last visited page

---

## State Management Split

| State type | Owner |
|------------|-------|
| Server data (agents, issues, companies) | React Query |
| Selected company ID | Zustand |
| Selected agent (detail panel open) | Zustand |
| WebSocket connection | Zustand (singleton) |
| Auth token | Zustand + localStorage sync |

---

## Project Structure

```
web/
  index.html
  package.json
  vite.config.ts
  tailwind.config.ts
  tsconfig.json
  src/
    main.tsx
    App.tsx                    # Router setup
    lib/
      api.ts                   # Axios instance
      queryClient.ts           # React Query client
    store/
      useAppStore.ts           # Zustand store (auth, selectedCompany, selectedAgent, ws)
    hooks/
      useWebSocket.ts          # WS connection + message dispatch
      useCompany.ts            # React Query hooks for company data
    components/
      Layout/
        Sidebar.tsx
        CompanySelector.tsx
      Dashboard/
        index.tsx
        AgentStatusCard.tsx
        EscalationList.tsx
      OrgChart/
        index.tsx
        AgentNode.tsx
        DetailPanel.tsx
      Issues/
        index.tsx
        IssueTable.tsx
        IssueDetail.tsx
      AgentChat/
        index.tsx              # Chat panel (used inside OrgChart DetailPanel)
      Hiring/
        index.tsx
        HireCard.tsx
      FSBrowser/
        index.tsx              # FS permissions table (used in DetailPanel)
    pages/
      Login.tsx
      Register.tsx
      CompanyList.tsx
      Company.tsx              # Shell that renders Sidebar + active page
```

---

## Build Integration

The Go server serves the built React app as a static SPA from `staticDir` (configured at startup). Vite builds to `web/dist`. The `handleSPA` handler in the API serves `index.html` for all non-API routes.

Development: `vite dev` proxies `/api/*` and `/ws` to `localhost:8080`.

---

## Out of Scope for Phase 4

- FS browser (shell exists, implementation deferred to Phase 5)
- Agent chat thread persistence UI (chat panel opens but full thread view is Phase 5)
- Multi-company switching (selector present but switching deferred)
- Mobile layout
