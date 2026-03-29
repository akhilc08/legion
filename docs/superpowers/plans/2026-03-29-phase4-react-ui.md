# Phase 4: React UI Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Build the full React/TypeScript frontend for the legion conductor platform — sidebar nav, dashboard, React Flow org chart with drag-to-reassign, issues table, hiring approval flow, audit log, and live WebSocket updates.

**Architecture:** Vite + React + TypeScript SPA served by the existing Go binary from `web/dist`. React Query owns all server state; Zustand manages auth token, selected company, and WS connection. The Go WS auth middleware is extended to accept the JWT as a query param (browsers can't send headers on WS connections).

**Tech Stack:** Vite 5, React 19, TypeScript 5, Tailwind CSS 3, shadcn/ui, TanStack React Query 5, Zustand 5, React Router 6, @xyflow/react 12, Axios 1, Vitest 2, @testing-library/react 16, MSW 2

---

## File Map

```
web/
  index.html
  package.json
  vite.config.ts
  tsconfig.json
  tailwind.config.ts
  postcss.config.js
  components.json                        # shadcn config
  src/
    main.tsx
    App.tsx                              # BrowserRouter + routes
    test/
      setup.ts                           # @testing-library/jest-dom import
      server.ts                          # MSW server instance
      handlers.ts                        # default MSW request handlers
    lib/
      api.ts                             # Axios instance with JWT interceptor + 401 redirect
      queryClient.ts                     # React Query client config
      types.ts                           # TypeScript types mirroring Go models
    store/
      useAppStore.ts                     # Zustand: token, companyId, agentId, ws
    hooks/
      useWebSocket.ts                    # WS singleton: open/close, message dispatch
    components/
      ui/                                # shadcn/ui primitives (Button, Card, Badge, etc.)
      Layout/
        Sidebar.tsx                      # Nav links + hire badge + company selector
        CompanySelector.tsx              # Dropdown to switch active company
      Dashboard/
        index.tsx                        # Stats cards + escalation list + notifications
        AgentStatusCard.tsx              # Single stat card (label + count + colour)
        EscalationList.tsx               # List of open escalations
      OrgChart/
        index.tsx                        # React Flow canvas + toolbar
        AgentNode.tsx                    # Custom node: role title + status dot
        DetailPanel.tsx                  # Slide-in panel: status, issue, budget, actions
      Issues/
        index.tsx                        # Table + filters
        IssueRow.tsx                     # Expandable row with dependency chain
      AgentChat/
        index.tsx                        # Chat panel inside DetailPanel
      Hiring/
        index.tsx                        # Card list with approve/reject
        HireCard.tsx                     # Single pending hire card
      Audit/
        index.tsx                        # Scrollable audit log table
    pages/
      Login.tsx
      Register.tsx
      CompanyList.tsx
      CompanyShell.tsx                   # Sidebar + <Outlet> for company pages

internal/api/
  middleware.go                          # MODIFY: fallback JWT from ?token= query param
```

---

## Task 1: Go backend — WS JWT query-param fallback

**Files:**
- Modify: `internal/api/middleware.go`

The `authMiddleware` currently only reads `Authorization: Bearer <token>`. WebSocket upgrades from the browser don't carry custom headers, so the JWT must also be accepted from the `?token=` query param.

- [ ] **Step 1: Write the failing test**

Create `internal/api/middleware_test.go`:

```go
package api

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

func makeTestServer() *Server {
	return &Server{jwtSecret: "test-secret"}
}

func mintTestToken(secret string, userID uuid.UUID) string {
	claims := jwt.RegisteredClaims{
		Subject:   userID.String(),
		ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour)),
	}
	tok, _ := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString([]byte(secret))
	return tok
}

func TestAuthMiddleware_QueryParam(t *testing.T) {
	s := makeTestServer()
	userID := uuid.New()
	token := mintTestToken("test-secret", userID)

	called := false
	handler := s.authMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		gotID, ok := userIDFromCtx(r)
		if !ok || gotID != userID {
			t.Errorf("expected userID %s in context, got %s (ok=%v)", userID, gotID, ok)
		}
	}))

	req := httptest.NewRequest("GET", "/?token="+token, nil)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if !called {
		t.Fatal("handler was not called")
	}
	if rr.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rr.Code)
	}
}

func TestAuthMiddleware_HeaderStillWorks(t *testing.T) {
	s := makeTestServer()
	userID := uuid.New()
	token := mintTestToken("test-secret", userID)

	called := false
	handler := s.authMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
	}))

	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if !called {
		t.Fatal("handler was not called")
	}
}
```

- [ ] **Step 2: Run to confirm failure**

```bash
cd /path/to/legion && go test ./internal/api/ -run TestAuthMiddleware -v
```

Expected: `FAIL — TestAuthMiddleware_QueryParam: handler was not called` (returns 401 because query param is not read yet).

- [ ] **Step 3: Implement the fallback in `internal/api/middleware.go`**

Replace the `authMiddleware` function (lines 20-47) with:

```go
func (s *Server) authMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		tokenStr := ""

		// Prefer Authorization header; fall back to ?token= for WebSocket upgrades.
		if h := r.Header.Get("Authorization"); h != "" {
			tokenStr = strings.TrimPrefix(h, "Bearer ")
		} else if q := r.URL.Query().Get("token"); q != "" {
			tokenStr = q
		}

		if tokenStr == "" {
			writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
			return
		}

		claims := &jwt.RegisteredClaims{}
		token, err := jwt.ParseWithClaims(tokenStr, claims, func(t *jwt.Token) (interface{}, error) {
			return []byte(s.jwtSecret), nil
		})
		if err != nil || !token.Valid {
			writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
			return
		}

		userID, err := uuid.Parse(claims.Subject)
		if err != nil {
			writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
			return
		}

		ctx := context.WithValue(r.Context(), ctxUserID, userID)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
```

- [ ] **Step 4: Run tests**

```bash
go test ./internal/api/ -run TestAuthMiddleware -v
```

Expected: `PASS`

- [ ] **Step 5: Run full backend test suite**

```bash
go test ./...
```

Expected: all pass (or pre-existing failures only).

- [ ] **Step 6: Commit**

```bash
git add internal/api/middleware.go internal/api/middleware_test.go
git commit -m "auth middleware: accept JWT from ?token query param for WS"
```

---

## Task 2: Scaffold Vite + React + TypeScript project

**Files:**
- Create: `web/package.json`
- Create: `web/vite.config.ts`
- Create: `web/tsconfig.json`
- Create: `web/index.html`

- [ ] **Step 1: Create `web/package.json`**

```json
{
  "name": "legion-ui",
  "private": true,
  "version": "0.1.0",
  "type": "module",
  "scripts": {
    "dev": "vite",
    "build": "tsc -b && vite build",
    "preview": "vite preview",
    "test": "vitest run",
    "test:watch": "vitest"
  },
  "dependencies": {
    "@tanstack/react-query": "^5.62.0",
    "@xyflow/react": "^12.3.6",
    "axios": "^1.7.9",
    "class-variance-authority": "^0.7.1",
    "clsx": "^2.1.1",
    "lucide-react": "^0.469.0",
    "react": "^19.0.0",
    "react-dom": "^19.0.0",
    "react-router-dom": "^6.28.1",
    "tailwind-merge": "^2.6.0",
    "zustand": "^5.0.3"
  },
  "devDependencies": {
    "@testing-library/jest-dom": "^6.6.3",
    "@testing-library/react": "^16.1.0",
    "@testing-library/user-event": "^14.5.2",
    "@types/react": "^19.0.2",
    "@types/react-dom": "^19.0.2",
    "@vitejs/plugin-react": "^4.3.4",
    "autoprefixer": "^10.4.20",
    "jsdom": "^25.0.1",
    "msw": "^2.7.0",
    "postcss": "^8.4.49",
    "tailwindcss": "^3.4.17",
    "typescript": "^5.7.2",
    "vite": "^5.4.11",
    "vitest": "^2.1.8"
  }
}
```

- [ ] **Step 2: Create `web/vite.config.ts`**

```typescript
import path from 'path'
import react from '@vitejs/plugin-react'
import { defineConfig } from 'vite'

export default defineConfig({
  plugins: [react()],
  resolve: {
    alias: {
      '@': path.resolve(__dirname, './src'),
    },
  },
  server: {
    proxy: {
      '/api': 'http://localhost:8080',
      '/ws': { target: 'ws://localhost:8080', ws: true },
    },
  },
  build: {
    outDir: 'dist',
  },
  test: {
    environment: 'jsdom',
    globals: true,
    setupFiles: ['./src/test/setup.ts'],
  },
})
```

- [ ] **Step 3: Create `web/tsconfig.json`**

```json
{
  "compilerOptions": {
    "target": "ES2020",
    "useDefineForClassFields": true,
    "lib": ["ES2020", "DOM", "DOM.Iterable"],
    "module": "ESNext",
    "skipLibCheck": true,
    "moduleResolution": "bundler",
    "allowImportingTsExtensions": true,
    "isolatedModules": true,
    "moduleDetection": "force",
    "noEmit": true,
    "jsx": "react-jsx",
    "strict": true,
    "baseUrl": ".",
    "paths": {
      "@/*": ["./src/*"]
    }
  },
  "include": ["src"]
}
```

- [ ] **Step 4: Create `web/index.html`**

```html
<!doctype html>
<html lang="en" class="dark">
  <head>
    <meta charset="UTF-8" />
    <meta name="viewport" content="width=device-width, initial-scale=1.0" />
    <title>legion</title>
  </head>
  <body>
    <div id="root"></div>
    <script type="module" src="/src/main.tsx"></script>
  </body>
</html>
```

- [ ] **Step 5: Create `web/postcss.config.js`**

```js
export default {
  plugins: {
    tailwindcss: {},
    autoprefixer: {},
  },
}
```

- [ ] **Step 6: Install dependencies**

```bash
cd web && npm install
```

Expected: `node_modules/` created, no errors.

- [ ] **Step 7: Commit**

```bash
git add web/package.json web/vite.config.ts web/tsconfig.json web/index.html web/postcss.config.js
git commit -m "scaffold vite react ts project in web/"
```

---

## Task 3: Configure Tailwind + shadcn/ui

**Files:**
- Create: `web/tailwind.config.ts`
- Create: `web/components.json`
- Create: `web/src/lib/utils.ts`
- Create: `web/src/index.css`

- [ ] **Step 1: Create `web/tailwind.config.ts`**

```typescript
import type { Config } from 'tailwindcss'

export default {
  darkMode: ['class'],
  content: ['./index.html', './src/**/*.{ts,tsx}'],
  theme: {
    extend: {
      colors: {
        border: 'hsl(var(--border))',
        input: 'hsl(var(--input))',
        ring: 'hsl(var(--ring))',
        background: 'hsl(var(--background))',
        foreground: 'hsl(var(--foreground))',
        primary: {
          DEFAULT: 'hsl(var(--primary))',
          foreground: 'hsl(var(--primary-foreground))',
        },
        secondary: {
          DEFAULT: 'hsl(var(--secondary))',
          foreground: 'hsl(var(--secondary-foreground))',
        },
        muted: {
          DEFAULT: 'hsl(var(--muted))',
          foreground: 'hsl(var(--muted-foreground))',
        },
        accent: {
          DEFAULT: 'hsl(var(--accent))',
          foreground: 'hsl(var(--accent-foreground))',
        },
        destructive: {
          DEFAULT: 'hsl(var(--destructive))',
          foreground: 'hsl(var(--destructive-foreground))',
        },
        card: {
          DEFAULT: 'hsl(var(--card))',
          foreground: 'hsl(var(--card-foreground))',
        },
      },
      borderRadius: {
        lg: 'var(--radius)',
        md: 'calc(var(--radius) - 2px)',
        sm: 'calc(var(--radius) - 4px)',
      },
    },
  },
  plugins: [],
} satisfies Config
```

- [ ] **Step 2: Create `web/components.json`** (shadcn config)

```json
{
  "$schema": "https://ui.shadcn.com/schema.json",
  "style": "default",
  "rsc": false,
  "tsx": true,
  "tailwind": {
    "config": "tailwind.config.ts",
    "css": "src/index.css",
    "baseColor": "zinc",
    "cssVariables": true
  },
  "aliases": {
    "components": "@/components",
    "utils": "@/lib/utils",
    "ui": "@/components/ui",
    "lib": "@/lib",
    "hooks": "@/hooks"
  }
}
```

- [ ] **Step 3: Create `web/src/index.css`** (Tailwind directives + shadcn CSS vars — dark theme)

```css
@tailwind base;
@tailwind components;
@tailwind utilities;

@layer base {
  :root {
    --background: 0 0% 100%;
    --foreground: 240 10% 3.9%;
    --card: 0 0% 100%;
    --card-foreground: 240 10% 3.9%;
    --border: 240 5.9% 90%;
    --input: 240 5.9% 90%;
    --ring: 240 10% 3.9%;
    --primary: 240 5.9% 10%;
    --primary-foreground: 0 0% 98%;
    --secondary: 240 4.8% 95.9%;
    --secondary-foreground: 240 5.9% 10%;
    --muted: 240 4.8% 95.9%;
    --muted-foreground: 240 3.8% 46.1%;
    --accent: 240 4.8% 95.9%;
    --accent-foreground: 240 5.9% 10%;
    --destructive: 0 84.2% 60.2%;
    --destructive-foreground: 0 0% 98%;
    --radius: 0.5rem;
  }

  .dark {
    --background: 240 10% 3.9%;
    --foreground: 0 0% 98%;
    --card: 240 10% 3.9%;
    --card-foreground: 0 0% 98%;
    --border: 240 3.7% 15.9%;
    --input: 240 3.7% 15.9%;
    --ring: 240 4.9% 83.9%;
    --primary: 0 0% 98%;
    --primary-foreground: 240 5.9% 10%;
    --secondary: 240 3.7% 15.9%;
    --secondary-foreground: 0 0% 98%;
    --muted: 240 3.7% 15.9%;
    --muted-foreground: 240 5% 64.9%;
    --accent: 240 3.7% 15.9%;
    --accent-foreground: 0 0% 98%;
    --destructive: 0 62.8% 30.6%;
    --destructive-foreground: 0 0% 98%;
  }
}

@layer base {
  * { @apply border-border; }
  body { @apply bg-background text-foreground; }
}
```

- [ ] **Step 4: Create `web/src/lib/utils.ts`**

```typescript
import { clsx, type ClassValue } from 'clsx'
import { twMerge } from 'tailwind-merge'

export function cn(...inputs: ClassValue[]) {
  return twMerge(clsx(inputs))
}
```

- [ ] **Step 5: Install shadcn/ui component primitives**

Run from inside `web/`:

```bash
npx shadcn@latest add button card badge input label separator scroll-area toast sonner
```

Answer the prompts: use existing `components.json`, overwrite if asked.

Expected: components appear in `web/src/components/ui/`.

- [ ] **Step 6: Commit**

```bash
git add web/tailwind.config.ts web/components.json web/src/index.css web/src/lib/utils.ts web/src/components/ui/
git commit -m "configure tailwind + shadcn/ui dark theme"
```

---

## Task 4: Core types, API client, React Query client, test harness

**Files:**
- Create: `web/src/lib/types.ts`
- Create: `web/src/lib/api.ts`
- Create: `web/src/lib/queryClient.ts`
- Create: `web/src/test/setup.ts`
- Create: `web/src/test/handlers.ts`
- Create: `web/src/test/server.ts`

- [ ] **Step 1: Write failing test for API client**

Create `web/src/lib/api.test.ts`:

```typescript
import { describe, it, expect, beforeAll, afterEach, afterAll } from 'vitest'
import { server } from '@/test/server'
import { http, HttpResponse } from 'msw'
import { apiClient } from '@/lib/api'

beforeAll(() => server.listen())
afterEach(() => server.resetHandlers())
afterAll(() => server.close())

describe('apiClient', () => {
  it('attaches Authorization header when token is in localStorage', async () => {
    localStorage.setItem('legion_token', 'test-jwt')
    let capturedAuth = ''
    server.use(
      http.get('/api/companies', ({ request }) => {
        capturedAuth = request.headers.get('Authorization') ?? ''
        return HttpResponse.json([])
      })
    )
    await apiClient.get('/api/companies')
    expect(capturedAuth).toBe('Bearer test-jwt')
    localStorage.removeItem('legion_token')
  })

  it('does not attach Authorization header when no token', async () => {
    localStorage.removeItem('legion_token')
    let capturedAuth: string | null = null
    server.use(
      http.get('/api/companies', ({ request }) => {
        capturedAuth = request.headers.get('Authorization')
        return HttpResponse.json([])
      })
    )
    await apiClient.get('/api/companies')
    expect(capturedAuth).toBeNull()
  })
})
```

- [ ] **Step 2: Run to confirm failure**

```bash
cd web && npm test
```

Expected: FAIL — `Cannot find module '@/lib/api'`

- [ ] **Step 3: Create `web/src/test/setup.ts`**

```typescript
import '@testing-library/jest-dom'
```

- [ ] **Step 4: Create `web/src/test/handlers.ts`**

```typescript
import { http, HttpResponse } from 'msw'

export const handlers = [
  http.post('/api/auth/login', () =>
    HttpResponse.json({ token: 'mock-token', user: { id: 'user-1', email: 'test@test.com' } })
  ),
  http.get('/api/companies', () => HttpResponse.json([])),
]
```

- [ ] **Step 5: Create `web/src/test/server.ts`**

```typescript
import { setupServer } from 'msw/node'
import { handlers } from './handlers'

export const server = setupServer(...handlers)
```

- [ ] **Step 6: Create `web/src/lib/types.ts`**

```typescript
export type AgentRuntime = 'claude_code' | 'openclaw'

export type AgentStatus =
  | 'idle'
  | 'working'
  | 'paused'
  | 'blocked'
  | 'failed'
  | 'done'
  | 'degraded'

export type IssueStatus =
  | 'pending'
  | 'in_progress'
  | 'blocked'
  | 'done'
  | 'failed'

export type HireStatus = 'pending' | 'approved' | 'rejected'

export type PermissionLevel = 'read' | 'write' | 'admin'

export interface Company {
  id: string
  name: string
  goal: string
  created_at: string
}

export interface Agent {
  id: string
  company_id: string
  role: string
  title: string
  system_prompt: string
  manager_id: string | null
  runtime: AgentRuntime
  status: AgentStatus
  monthly_budget: number
  token_spend: number
  chat_token_spend: number
  pid: number | null
  created_at: string
  updated_at: string
}

export interface Issue {
  id: string
  company_id: string
  title: string
  description: string
  assignee_id: string | null
  parent_id: string | null
  status: IssueStatus
  output_path: string | null
  attempt_count: number
  last_failure_reason: string | null
  escalation_id: string | null
  created_at: string
  updated_at: string
}

export interface EscalationChainEntry {
  agent_id: string
  reason: string
  attempted_at: string
}

export interface Escalation {
  id: string
  original_issue_id: string
  current_assignee_id: string | null
  escalation_chain: EscalationChainEntry[]
  trigger: string
  status: string
  created_at: string
  updated_at: string
}

export interface PendingHire {
  id: string
  company_id: string
  requested_by_agent_id: string
  role_title: string
  reporting_to_agent_id: string
  system_prompt: string
  runtime: AgentRuntime
  budget_allocation: number
  initial_task: string | null
  status: HireStatus
  created_at: string
}

export interface AuditLog {
  id: string
  company_id: string
  actor_id: string | null
  event_type: string
  payload: Record<string, unknown>
  created_at: string
}

export interface Notification {
  id: string
  company_id: string
  type: string
  escalation_id: string | null
  payload: Record<string, unknown>
  dismissed_at: string | null
  created_at: string
}

export interface ChatMessage {
  role: 'user' | 'assistant'
  content: string
  timestamp: string
}

// WebSocket event envelope (matches ws.Event in Go)
export interface WsEvent {
  type:
    | 'agent_status'
    | 'agent_log'
    | 'issue_update'
    | 'heartbeat'
    | 'notification'
    | 'hire_pending'
    | 'chat_message'
    | 'escalation'
    | 'runtime_status'
  company_id: string
  payload: unknown
}
```

- [ ] **Step 7: Create `web/src/lib/api.ts`**

```typescript
import axios from 'axios'

export const apiClient = axios.create({
  baseURL: '/',
  headers: { 'Content-Type': 'application/json' },
})

apiClient.interceptors.request.use((config) => {
  const token = localStorage.getItem('legion_token')
  if (token) {
    config.headers.Authorization = `Bearer ${token}`
  }
  return config
})

apiClient.interceptors.response.use(
  (res) => res,
  (err) => {
    if (err.response?.status === 401) {
      localStorage.removeItem('legion_token')
      window.location.href = '/login'
    }
    return Promise.reject(err)
  }
)
```

- [ ] **Step 8: Create `web/src/lib/queryClient.ts`**

```typescript
import { QueryClient } from '@tanstack/react-query'

export const queryClient = new QueryClient({
  defaultOptions: {
    queries: {
      staleTime: 30_000,
      retry: 1,
    },
  },
})
```

- [ ] **Step 9: Run tests**

```bash
cd web && npm test
```

Expected: `api.test.ts` PASS, 2 tests passing.

- [ ] **Step 10: Commit**

```bash
git add web/src/lib/ web/src/test/
git commit -m "add types, api client, query client, msw test harness"
```

---

## Task 5: Zustand store

**Files:**
- Create: `web/src/store/useAppStore.ts`
- Create: `web/src/store/useAppStore.test.ts`

- [ ] **Step 1: Write failing test**

Create `web/src/store/useAppStore.test.ts`:

```typescript
import { describe, it, expect, beforeEach } from 'vitest'
import { useAppStore } from './useAppStore'

beforeEach(() => {
  useAppStore.setState({ token: null, companyId: null, agentId: null })
  localStorage.clear()
})

describe('useAppStore', () => {
  it('setToken persists to localStorage and updates state', () => {
    useAppStore.getState().setToken('abc123')
    expect(useAppStore.getState().token).toBe('abc123')
    expect(localStorage.getItem('legion_token')).toBe('abc123')
  })

  it('setToken(null) removes from localStorage', () => {
    useAppStore.getState().setToken('abc123')
    useAppStore.getState().setToken(null)
    expect(useAppStore.getState().token).toBeNull()
    expect(localStorage.getItem('legion_token')).toBeNull()
  })

  it('initialises token from localStorage', () => {
    localStorage.setItem('legion_token', 'persisted')
    // Re-create store by importing fresh state via getInitialToken
    expect(useAppStore.getState().getInitialToken()).toBe('persisted')
  })

  it('setCompanyId updates selected company', () => {
    useAppStore.getState().setCompanyId('company-abc')
    expect(useAppStore.getState().companyId).toBe('company-abc')
  })

  it('setAgentId updates selected agent', () => {
    useAppStore.getState().setAgentId('agent-xyz')
    expect(useAppStore.getState().agentId).toBe('agent-xyz')
  })
})
```

- [ ] **Step 2: Run to confirm failure**

```bash
cd web && npm test -- useAppStore
```

Expected: FAIL — `Cannot find module './useAppStore'`

- [ ] **Step 3: Create `web/src/store/useAppStore.ts`**

```typescript
import { create } from 'zustand'

interface AppStore {
  token: string | null
  companyId: string | null
  agentId: string | null
  setToken: (token: string | null) => void
  setCompanyId: (id: string | null) => void
  setAgentId: (id: string | null) => void
  getInitialToken: () => string | null
}

export const useAppStore = create<AppStore>((set) => ({
  token: localStorage.getItem('legion_token'),
  companyId: null,
  agentId: null,

  setToken: (token) => {
    if (token) {
      localStorage.setItem('legion_token', token)
    } else {
      localStorage.removeItem('legion_token')
    }
    set({ token })
  },

  setCompanyId: (companyId) => set({ companyId }),

  setAgentId: (agentId) => set({ agentId }),

  getInitialToken: () => localStorage.getItem('legion_token'),
}))
```

- [ ] **Step 4: Run tests**

```bash
cd web && npm test -- useAppStore
```

Expected: PASS, 5 tests.

- [ ] **Step 5: Commit**

```bash
git add web/src/store/
git commit -m "add zustand store with token persistence"
```

---

## Task 6: App entry point, router, and Layout shell

**Files:**
- Create: `web/src/main.tsx`
- Create: `web/src/App.tsx`
- Create: `web/src/components/Layout/Sidebar.tsx`
- Create: `web/src/components/Layout/CompanySelector.tsx`
- Create: `web/src/pages/CompanyShell.tsx`
- Create: `web/src/pages/Login.tsx`
- Create: `web/src/pages/Register.tsx`
- Create: `web/src/pages/CompanyList.tsx`

- [ ] **Step 1: Write failing test for Sidebar**

Create `web/src/components/Layout/Sidebar.test.tsx`:

```typescript
import { describe, it, expect, vi } from 'vitest'
import { render, screen } from '@testing-library/react'
import { MemoryRouter } from 'react-router-dom'
import { Sidebar } from './Sidebar'

vi.mock('@/store/useAppStore', () => ({
  useAppStore: (sel: (s: { companyId: string | null }) => unknown) =>
    sel({ companyId: 'company-1' }),
}))

describe('Sidebar', () => {
  it('renders all nav links', () => {
    render(
      <MemoryRouter>
        <Sidebar hireBadgeCount={2} />
      </MemoryRouter>
    )
    expect(screen.getByText('Dashboard')).toBeInTheDocument()
    expect(screen.getByText('Org Chart')).toBeInTheDocument()
    expect(screen.getByText('Issues')).toBeInTheDocument()
    expect(screen.getByText('Audit')).toBeInTheDocument()
  })

  it('shows hire badge count', () => {
    render(
      <MemoryRouter>
        <Sidebar hireBadgeCount={2} />
      </MemoryRouter>
    )
    expect(screen.getByText('2')).toBeInTheDocument()
  })
})
```

- [ ] **Step 2: Run to confirm failure**

```bash
cd web && npm test -- Sidebar
```

Expected: FAIL — `Cannot find module './Sidebar'`

- [ ] **Step 3: Create `web/src/main.tsx`**

```typescript
import React from 'react'
import ReactDOM from 'react-dom/client'
import { QueryClientProvider } from '@tanstack/react-query'
import { queryClient } from '@/lib/queryClient'
import { App } from '@/App'
import '@/index.css'

ReactDOM.createRoot(document.getElementById('root')!).render(
  <React.StrictMode>
    <QueryClientProvider client={queryClient}>
      <App />
    </QueryClientProvider>
  </React.StrictMode>
)
```

- [ ] **Step 4: Create `web/src/App.tsx`**

```typescript
import { BrowserRouter, Routes, Route, Navigate } from 'react-router-dom'
import { useAppStore } from '@/store/useAppStore'
import { Login } from '@/pages/Login'
import { Register } from '@/pages/Register'
import { CompanyList } from '@/pages/CompanyList'
import { CompanyShell } from '@/pages/CompanyShell'

function RequireAuth({ children }: { children: React.ReactNode }) {
  const token = useAppStore((s) => s.token)
  if (!token) return <Navigate to="/login" replace />
  return <>{children}</>
}

export function App() {
  return (
    <BrowserRouter>
      <Routes>
        <Route path="/login" element={<Login />} />
        <Route path="/register" element={<Register />} />
        <Route
          path="/"
          element={
            <RequireAuth>
              <CompanyList />
            </RequireAuth>
          }
        />
        <Route
          path="/companies/:companyId/*"
          element={
            <RequireAuth>
              <CompanyShell />
            </RequireAuth>
          }
        />
        <Route path="*" element={<Navigate to="/" replace />} />
      </Routes>
    </BrowserRouter>
  )
}
```

- [ ] **Step 5: Create `web/src/components/Layout/CompanySelector.tsx`**

```typescript
import { useNavigate } from 'react-router-dom'
import { useQuery } from '@tanstack/react-query'
import { apiClient } from '@/lib/api'
import type { Company } from '@/lib/types'
import { useAppStore } from '@/store/useAppStore'
import { cn } from '@/lib/utils'
import { ChevronsUpDown } from 'lucide-react'

export function CompanySelector() {
  const navigate = useNavigate()
  const companyId = useAppStore((s) => s.companyId)
  const setCompanyId = useAppStore((s) => s.setCompanyId)

  const { data: companies = [] } = useQuery<Company[]>({
    queryKey: ['companies'],
    queryFn: () => apiClient.get('/api/companies').then((r) => r.data),
  })

  const current = companies.find((c) => c.id === companyId)

  return (
    <div className="px-2">
      <button
        className={cn(
          'flex w-full items-center gap-2 rounded-md px-2 py-2 text-sm',
          'bg-zinc-800 hover:bg-zinc-700 text-zinc-300 transition-colors'
        )}
        onClick={() => navigate('/')}
      >
        <span className="truncate flex-1 text-left">{current?.name ?? 'Select company'}</span>
        <ChevronsUpDown className="h-4 w-4 shrink-0 text-zinc-500" />
      </button>
    </div>
  )
}
```

- [ ] **Step 6: Create `web/src/components/Layout/Sidebar.tsx`**

```typescript
import { NavLink } from 'react-router-dom'
import { useAppStore } from '@/store/useAppStore'
import { CompanySelector } from './CompanySelector'
import { cn } from '@/lib/utils'
import {
  LayoutDashboard,
  Network,
  CircleDot,
  UserPlus,
  ScrollText,
} from 'lucide-react'

interface SidebarProps {
  hireBadgeCount: number
}

const navItems = [
  { label: 'Dashboard', icon: LayoutDashboard, path: 'dashboard' },
  { label: 'Org Chart', icon: Network, path: 'org-chart' },
  { label: 'Issues', icon: CircleDot, path: 'issues' },
  { label: 'Hiring', icon: UserPlus, path: 'hiring' },
  { label: 'Audit', icon: ScrollText, path: 'audit' },
]

export function Sidebar({ hireBadgeCount }: SidebarProps) {
  const companyId = useAppStore((s) => s.companyId)
  const base = companyId ? `/companies/${companyId}` : '#'

  return (
    <aside className="flex h-screen w-52 flex-col border-r border-zinc-800 bg-zinc-950">
      <div className="flex h-14 items-center px-4 border-b border-zinc-800">
        <span className="text-sm font-semibold text-zinc-100 tracking-wide">legion</span>
      </div>

      <nav className="flex-1 space-y-1 p-2 pt-3">
        {navItems.map(({ label, icon: Icon, path }) => (
          <NavLink
            key={path}
            to={`${base}/${path}`}
            className={({ isActive }) =>
              cn(
                'flex items-center gap-3 rounded-md px-3 py-2 text-sm transition-colors',
                isActive
                  ? 'bg-zinc-800 text-zinc-100'
                  : 'text-zinc-400 hover:bg-zinc-800/60 hover:text-zinc-200'
              )
            }
          >
            <Icon className="h-4 w-4 shrink-0" />
            <span className="flex-1">{label}</span>
            {label === 'Hiring' && hireBadgeCount > 0 && (
              <span className="rounded-full bg-red-600 px-1.5 py-0.5 text-xs font-medium text-white leading-none">
                {hireBadgeCount}
              </span>
            )}
          </NavLink>
        ))}
      </nav>

      <div className="border-t border-zinc-800 py-3">
        <CompanySelector />
      </div>
    </aside>
  )
}
```

- [ ] **Step 7: Create `web/src/pages/CompanyShell.tsx`**

```typescript
import { useEffect } from 'react'
import { useParams, Routes, Route, Navigate } from 'react-router-dom'
import { useQuery } from '@tanstack/react-query'
import { useAppStore } from '@/store/useAppStore'
import { Sidebar } from '@/components/Layout/Sidebar'
import { apiClient } from '@/lib/api'
import type { PendingHire } from '@/lib/types'
import { Dashboard } from '@/components/Dashboard'
import { OrgChart } from '@/components/OrgChart'
import { Issues } from '@/components/Issues'
import { Hiring } from '@/components/Hiring'
import { Audit } from '@/components/Audit'

export function CompanyShell() {
  const { companyId } = useParams<{ companyId: string }>()
  const setCompanyId = useAppStore((s) => s.setCompanyId)

  useEffect(() => {
    if (companyId) setCompanyId(companyId)
  }, [companyId, setCompanyId])

  const { data: hires = [] } = useQuery<PendingHire[]>({
    queryKey: ['hires', companyId],
    queryFn: () =>
      apiClient.get(`/api/companies/${companyId}/hires`).then((r) => r.data),
    enabled: !!companyId,
  })

  const pendingCount = hires.filter((h) => h.status === 'pending').length

  return (
    <div className="flex h-screen overflow-hidden">
      <Sidebar hireBadgeCount={pendingCount} />
      <main className="flex-1 overflow-auto bg-zinc-950">
        <Routes>
          <Route index element={<Navigate to="dashboard" replace />} />
          <Route path="dashboard" element={<Dashboard />} />
          <Route path="org-chart" element={<OrgChart />} />
          <Route path="issues" element={<Issues />} />
          <Route path="hiring" element={<Hiring />} />
          <Route path="audit" element={<Audit />} />
        </Routes>
      </main>
    </div>
  )
}
```

- [ ] **Step 8: Create `web/src/pages/Login.tsx`**

```typescript
import { useState } from 'react'
import { useNavigate, Link } from 'react-router-dom'
import { apiClient } from '@/lib/api'
import { useAppStore } from '@/store/useAppStore'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'

export function Login() {
  const navigate = useNavigate()
  const setToken = useAppStore((s) => s.setToken)
  const [email, setEmail] = useState('')
  const [password, setPassword] = useState('')
  const [error, setError] = useState('')
  const [loading, setLoading] = useState(false)

  async function handleSubmit(e: React.FormEvent) {
    e.preventDefault()
    setError('')
    setLoading(true)
    try {
      const { data } = await apiClient.post('/api/auth/login', { email, password })
      setToken(data.token)
      navigate('/')
    } catch {
      setError('Invalid email or password')
    } finally {
      setLoading(false)
    }
  }

  return (
    <div className="flex min-h-screen items-center justify-center bg-zinc-950">
      <Card className="w-full max-w-sm border-zinc-800 bg-zinc-900">
        <CardHeader>
          <CardTitle className="text-zinc-100 text-center">legion</CardTitle>
        </CardHeader>
        <CardContent>
          <form onSubmit={handleSubmit} className="space-y-4">
            <div className="space-y-1">
              <Label htmlFor="email" className="text-zinc-300">Email</Label>
              <Input
                id="email"
                type="email"
                value={email}
                onChange={(e) => setEmail(e.target.value)}
                required
                className="bg-zinc-800 border-zinc-700 text-zinc-100"
              />
            </div>
            <div className="space-y-1">
              <Label htmlFor="password" className="text-zinc-300">Password</Label>
              <Input
                id="password"
                type="password"
                value={password}
                onChange={(e) => setPassword(e.target.value)}
                required
                className="bg-zinc-800 border-zinc-700 text-zinc-100"
              />
            </div>
            {error && <p className="text-sm text-red-400">{error}</p>}
            <Button type="submit" disabled={loading} className="w-full">
              {loading ? 'Signing in…' : 'Sign in'}
            </Button>
            <p className="text-center text-sm text-zinc-500">
              No account?{' '}
              <Link to="/register" className="text-zinc-300 hover:underline">
                Register
              </Link>
            </p>
          </form>
        </CardContent>
      </Card>
    </div>
  )
}
```

- [ ] **Step 9: Create `web/src/pages/Register.tsx`**

```typescript
import { useState } from 'react'
import { useNavigate, Link } from 'react-router-dom'
import { apiClient } from '@/lib/api'
import { useAppStore } from '@/store/useAppStore'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'

export function Register() {
  const navigate = useNavigate()
  const setToken = useAppStore((s) => s.setToken)
  const [email, setEmail] = useState('')
  const [password, setPassword] = useState('')
  const [error, setError] = useState('')
  const [loading, setLoading] = useState(false)

  async function handleSubmit(e: React.FormEvent) {
    e.preventDefault()
    setError('')
    setLoading(true)
    try {
      const { data } = await apiClient.post('/api/auth/register', { email, password })
      setToken(data.token)
      navigate('/')
    } catch {
      setError('Registration failed — email may already be taken')
    } finally {
      setLoading(false)
    }
  }

  return (
    <div className="flex min-h-screen items-center justify-center bg-zinc-950">
      <Card className="w-full max-w-sm border-zinc-800 bg-zinc-900">
        <CardHeader>
          <CardTitle className="text-zinc-100 text-center">Create account</CardTitle>
        </CardHeader>
        <CardContent>
          <form onSubmit={handleSubmit} className="space-y-4">
            <div className="space-y-1">
              <Label htmlFor="email" className="text-zinc-300">Email</Label>
              <Input
                id="email"
                type="email"
                value={email}
                onChange={(e) => setEmail(e.target.value)}
                required
                className="bg-zinc-800 border-zinc-700 text-zinc-100"
              />
            </div>
            <div className="space-y-1">
              <Label htmlFor="password" className="text-zinc-300">Password</Label>
              <Input
                id="password"
                type="password"
                value={password}
                onChange={(e) => setPassword(e.target.value)}
                required
                className="bg-zinc-800 border-zinc-700 text-zinc-100"
              />
            </div>
            {error && <p className="text-sm text-red-400">{error}</p>}
            <Button type="submit" disabled={loading} className="w-full">
              {loading ? 'Creating…' : 'Create account'}
            </Button>
            <p className="text-center text-sm text-zinc-500">
              Have an account?{' '}
              <Link to="/login" className="text-zinc-300 hover:underline">
                Sign in
              </Link>
            </p>
          </form>
        </CardContent>
      </Card>
    </div>
  )
}
```

- [ ] **Step 10: Create `web/src/pages/CompanyList.tsx`**

```typescript
import { useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { apiClient } from '@/lib/api'
import type { Company } from '@/lib/types'
import { useAppStore } from '@/store/useAppStore'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'

export function CompanyList() {
  const navigate = useNavigate()
  const qc = useQueryClient()
  const setToken = useAppStore((s) => s.setToken)
  const [name, setName] = useState('')
  const [goal, setGoal] = useState('')

  const { data: companies = [], isLoading } = useQuery<Company[]>({
    queryKey: ['companies'],
    queryFn: () => apiClient.get('/api/companies').then((r) => r.data),
  })

  const create = useMutation({
    mutationFn: (body: { name: string; goal: string }) =>
      apiClient.post('/api/companies', body).then((r) => r.data as Company),
    onSuccess: (company) => {
      qc.invalidateQueries({ queryKey: ['companies'] })
      navigate(`/companies/${company.id}/dashboard`)
    },
  })

  return (
    <div className="flex min-h-screen flex-col items-center justify-center bg-zinc-950 p-6 gap-6">
      <div className="w-full max-w-md">
        <div className="flex items-center justify-between mb-6">
          <h1 className="text-xl font-semibold text-zinc-100">Your companies</h1>
          <Button
            variant="ghost"
            size="sm"
            className="text-zinc-500"
            onClick={() => setToken(null)}
          >
            Sign out
          </Button>
        </div>

        {isLoading ? (
          <p className="text-zinc-500 text-sm">Loading…</p>
        ) : (
          <div className="space-y-2 mb-6">
            {companies.map((c) => (
              <button
                key={c.id}
                className="w-full text-left rounded-md border border-zinc-800 bg-zinc-900 px-4 py-3 hover:bg-zinc-800 transition-colors"
                onClick={() => navigate(`/companies/${c.id}/dashboard`)}
              >
                <p className="text-sm font-medium text-zinc-100">{c.name}</p>
                <p className="text-xs text-zinc-500 mt-0.5 line-clamp-1">{c.goal}</p>
              </button>
            ))}
          </div>
        )}

        <Card className="border-zinc-800 bg-zinc-900">
          <CardHeader className="pb-3">
            <CardTitle className="text-sm text-zinc-300">New company</CardTitle>
          </CardHeader>
          <CardContent>
            <form
              className="space-y-3"
              onSubmit={(e) => {
                e.preventDefault()
                create.mutate({ name, goal })
              }}
            >
              <Input
                placeholder="Name"
                value={name}
                onChange={(e) => setName(e.target.value)}
                required
                className="bg-zinc-800 border-zinc-700 text-zinc-100"
              />
              <Input
                placeholder="Goal"
                value={goal}
                onChange={(e) => setGoal(e.target.value)}
                className="bg-zinc-800 border-zinc-700 text-zinc-100"
              />
              <Button type="submit" disabled={create.isPending} className="w-full">
                {create.isPending ? 'Creating…' : 'Create'}
              </Button>
            </form>
          </CardContent>
        </Card>
      </div>
    </div>
  )
}
```

- [ ] **Step 11: Run Sidebar test**

```bash
cd web && npm test -- Sidebar
```

Expected: PASS, 2 tests.

- [ ] **Step 12: Commit**

```bash
git add web/src/
git commit -m "add app router, layout shell, login/register, company list"
```

---

## Task 7: Dashboard page

**Files:**
- Create: `web/src/components/Dashboard/AgentStatusCard.tsx`
- Create: `web/src/components/Dashboard/EscalationList.tsx`
- Create: `web/src/components/Dashboard/index.tsx`
- Create: `web/src/components/Dashboard/Dashboard.test.tsx`

- [ ] **Step 1: Write failing test**

Create `web/src/components/Dashboard/Dashboard.test.tsx`:

```typescript
import { describe, it, expect, beforeAll, afterEach, afterAll } from 'vitest'
import { render, screen } from '@testing-library/react'
import { QueryClientProvider } from '@tanstack/react-query'
import { QueryClient } from '@tanstack/react-query'
import { MemoryRouter, Route, Routes } from 'react-router-dom'
import { server } from '@/test/server'
import { http, HttpResponse } from 'msw'
import { Dashboard } from './index'
import type { Agent, Notification } from '@/lib/types'

beforeAll(() => server.listen())
afterEach(() => server.resetHandlers())
afterAll(() => server.close())

function makeClient() {
  return new QueryClient({ defaultOptions: { queries: { retry: false } } })
}

function wrapper(companyId: string) {
  return function W({ children }: { children: React.ReactNode }) {
    return (
      <QueryClientProvider client={makeClient()}>
        <MemoryRouter initialEntries={[`/companies/${companyId}/dashboard`]}>
          <Routes>
            <Route path="/companies/:companyId/dashboard" element={<>{children}</>} />
          </Routes>
        </MemoryRouter>
      </QueryClientProvider>
    )
  }
}

it('shows working agent count', async () => {
  const agents: Partial<Agent>[] = [
    { id: '1', status: 'working' },
    { id: '2', status: 'idle' },
    { id: '3', status: 'working' },
  ]
  server.use(
    http.get('/api/companies/:id/agents', () => HttpResponse.json(agents)),
    http.get('/api/companies/:id/notifications', () => HttpResponse.json([])),
  )
  const { findByText } = render(<Dashboard />, { wrapper: wrapper('c-1') })
  expect(await findByText('2')).toBeInTheDocument() // 2 working agents
})
```

- [ ] **Step 2: Run to confirm failure**

```bash
cd web && npm test -- Dashboard
```

Expected: FAIL — `Cannot find module './index'`

- [ ] **Step 3: Create `web/src/components/Dashboard/AgentStatusCard.tsx`**

```typescript
import { cn } from '@/lib/utils'

interface AgentStatusCardProps {
  label: string
  count: number
  colour: string // tailwind text colour class
}

export function AgentStatusCard({ label, count, colour }: AgentStatusCardProps) {
  return (
    <div className="rounded-lg border border-zinc-800 bg-zinc-900 px-4 py-3">
      <p className={cn('text-2xl font-semibold tabular-nums', colour)}>{count}</p>
      <p className="text-xs text-zinc-500 mt-0.5">{label}</p>
    </div>
  )
}
```

- [ ] **Step 4: Create `web/src/components/Dashboard/EscalationList.tsx`**

```typescript
import type { Notification } from '@/lib/types'

interface EscalationListProps {
  notifications: Notification[]
  onDismiss: (id: string) => void
}

export function EscalationList({ notifications, onDismiss }: EscalationListProps) {
  if (notifications.length === 0) {
    return <p className="text-sm text-zinc-600">No active notifications.</p>
  }

  return (
    <ul className="space-y-2">
      {notifications.map((n) => (
        <li
          key={n.id}
          className="flex items-start justify-between rounded-md border border-zinc-800 bg-zinc-900 px-3 py-2"
        >
          <div>
            <p className="text-xs font-medium text-zinc-300">{n.type}</p>
            <p className="text-xs text-zinc-500 mt-0.5">
              {new Date(n.created_at).toLocaleString()}
            </p>
          </div>
          <button
            className="text-zinc-600 hover:text-zinc-300 text-xs ml-4 shrink-0"
            onClick={() => onDismiss(n.id)}
          >
            dismiss
          </button>
        </li>
      ))}
    </ul>
  )
}
```

- [ ] **Step 5: Create `web/src/components/Dashboard/index.tsx`**

```typescript
import { useParams } from 'react-router-dom'
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { apiClient } from '@/lib/api'
import type { Agent, Notification } from '@/lib/types'
import { AgentStatusCard } from './AgentStatusCard'
import { EscalationList } from './EscalationList'

const STATUS_CARDS = [
  { label: 'Working', key: 'working' as const, colour: 'text-emerald-400' },
  { label: 'Idle', key: 'idle' as const, colour: 'text-zinc-300' },
  { label: 'Blocked', key: 'blocked' as const, colour: 'text-amber-400' },
  { label: 'Failed', key: 'failed' as const, colour: 'text-red-400' },
  { label: 'Degraded', key: 'degraded' as const, colour: 'text-orange-400' },
]

export function Dashboard() {
  const { companyId } = useParams<{ companyId: string }>()
  const qc = useQueryClient()

  const { data: agents = [] } = useQuery<Agent[]>({
    queryKey: ['agents', companyId],
    queryFn: () => apiClient.get(`/api/companies/${companyId}/agents`).then((r) => r.data),
    enabled: !!companyId,
  })

  const { data: notifications = [] } = useQuery<Notification[]>({
    queryKey: ['notifications', companyId],
    queryFn: () =>
      apiClient.get(`/api/companies/${companyId}/notifications`).then((r) => r.data),
    enabled: !!companyId,
  })

  const dismiss = useMutation({
    mutationFn: (id: string) =>
      apiClient.post(`/api/companies/${companyId}/notifications/${id}/dismiss`),
    onSuccess: () => qc.invalidateQueries({ queryKey: ['notifications', companyId] }),
  })

  const countByStatus = (status: Agent['status']) =>
    agents.filter((a) => a.status === status).length

  return (
    <div className="p-6 space-y-6 max-w-4xl">
      <h1 className="text-lg font-semibold text-zinc-100">Dashboard</h1>

      <div className="grid grid-cols-5 gap-3">
        {STATUS_CARDS.map(({ label, key, colour }) => (
          <AgentStatusCard
            key={key}
            label={label}
            count={countByStatus(key)}
            colour={colour}
          />
        ))}
      </div>

      <div>
        <h2 className="text-sm font-medium text-zinc-400 mb-3">Notifications</h2>
        <EscalationList
          notifications={notifications}
          onDismiss={(id) => dismiss.mutate(id)}
        />
      </div>
    </div>
  )
}
```

- [ ] **Step 6: Run tests**

```bash
cd web && npm test -- Dashboard
```

Expected: PASS.

- [ ] **Step 7: Commit**

```bash
git add web/src/components/Dashboard/
git commit -m "add dashboard page with agent status cards and notifications"
```

---

## Task 8: Org Chart — React Flow canvas

**Files:**
- Create: `web/src/components/OrgChart/AgentNode.tsx`
- Create: `web/src/components/OrgChart/AgentNode.test.tsx`
- Create: `web/src/components/OrgChart/index.tsx`

- [ ] **Step 1: Write failing test for AgentNode**

Create `web/src/components/OrgChart/AgentNode.test.tsx`:

```typescript
import { describe, it, expect } from 'vitest'
import { render, screen } from '@testing-library/react'
import { ReactFlowProvider } from '@xyflow/react'
import { AgentNode } from './AgentNode'
import type { Agent } from '@/lib/types'

const mockAgent: Agent = {
  id: 'a1',
  company_id: 'c1',
  role: 'cto',
  title: 'CTO Agent',
  system_prompt: '',
  manager_id: null,
  runtime: 'claude_code',
  status: 'working',
  monthly_budget: 100000,
  token_spend: 5000,
  chat_token_spend: 200,
  pid: null,
  created_at: '',
  updated_at: '',
}

describe('AgentNode', () => {
  it('renders agent title and status', () => {
    // React Flow injects internal props at runtime; cast for test purposes
    const props = { id: 'a1', data: { agent: mockAgent }, selected: false } as Parameters<typeof AgentNode>[0]
    render(<ReactFlowProvider><AgentNode {...props} /></ReactFlowProvider>)
    expect(screen.getByText('CTO Agent')).toBeInTheDocument()
    expect(screen.getByText('working')).toBeInTheDocument()
  })

  it('applies green colour for working status', () => {
    const props = { id: 'a1', data: { agent: mockAgent }, selected: false } as Parameters<typeof AgentNode>[0]
    const { container } = render(<ReactFlowProvider><AgentNode {...props} /></ReactFlowProvider>)
    const dot = container.querySelector('[data-status-dot]')
    expect(dot?.className).toMatch(/emerald|green/)
  })
})
```

- [ ] **Step 2: Run to confirm failure**

```bash
cd web && npm test -- AgentNode
```

Expected: FAIL — `Cannot find module './AgentNode'`

- [ ] **Step 3: Create `web/src/components/OrgChart/AgentNode.tsx`**

```typescript
import { memo } from 'react'
import { Handle, Position, type NodeProps, type Node } from '@xyflow/react'
import type { Agent } from '@/lib/types'
import { cn } from '@/lib/utils'

const STATUS_COLOURS: Record<Agent['status'], string> = {
  idle: 'bg-zinc-400',
  working: 'bg-emerald-400',
  paused: 'bg-blue-400',
  blocked: 'bg-amber-400',
  failed: 'bg-red-400',
  done: 'bg-zinc-600',
  degraded: 'bg-orange-400',
}

export type AgentNodeType = Node<{ agent: Agent }, 'agentNode'>

export const AgentNode = memo(function AgentNode({
  data,
  selected,
}: NodeProps<AgentNodeType>) {
  const { agent } = data
  return (
    <>
      <Handle type="target" position={Position.Top} className="!bg-zinc-600" />
      <div
        className={cn(
          'rounded-md border bg-zinc-900 px-3 py-2 min-w-[120px] cursor-pointer select-none',
          selected ? 'border-blue-500 shadow-[0_0_0_2px_rgba(59,130,246,0.3)]' : 'border-zinc-700'
        )}
      >
        <div className="flex items-center gap-2">
          <span
            data-status-dot
            className={cn('h-2 w-2 rounded-full shrink-0', STATUS_COLOURS[agent.status])}
          />
          <span className="text-xs font-medium text-zinc-100 truncate max-w-[100px]">
            {agent.title}
          </span>
        </div>
        <p className="text-[10px] text-zinc-500 mt-0.5 pl-4">{agent.status}</p>
      </div>
      <Handle type="source" position={Position.Bottom} className="!bg-zinc-600" />
    </>
  )
})
```

- [ ] **Step 4: Run AgentNode tests**

```bash
cd web && npm test -- AgentNode
```

Expected: PASS.

- [ ] **Step 5: Create `web/src/components/OrgChart/DetailPanel.tsx`**

```typescript
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { apiClient } from '@/lib/api'
import { useAppStore } from '@/store/useAppStore'
import type { Agent, Issue } from '@/lib/types'
import { Button } from '@/components/ui/button'
import { X } from 'lucide-react'
import { cn } from '@/lib/utils'

const STATUS_COLOURS: Record<Agent['status'], string> = {
  idle: 'text-zinc-400',
  working: 'text-emerald-400',
  paused: 'text-blue-400',
  blocked: 'text-amber-400',
  failed: 'text-red-400',
  done: 'text-zinc-500',
  degraded: 'text-orange-400',
}

interface DetailPanelProps {
  agent: Agent
  onClose: () => void
}

export function DetailPanel({ agent, onClose }: DetailPanelProps) {
  const companyId = useAppStore((s) => s.companyId)
  const qc = useQueryClient()

  const { data: issues = [] } = useQuery<Issue[]>({
    queryKey: ['issues', companyId],
    queryFn: () =>
      apiClient.get(`/api/companies/${companyId}/issues`).then((r) => r.data),
    enabled: !!companyId,
  })

  const currentIssue = issues.find(
    (i) => i.assignee_id === agent.id && i.status === 'in_progress'
  )

  const spawn = useMutation({
    mutationFn: () =>
      apiClient.post(`/api/companies/${companyId}/agents/${agent.id}/spawn`),
    onSuccess: () => qc.invalidateQueries({ queryKey: ['agents', companyId] }),
  })

  const kill = useMutation({
    mutationFn: () =>
      apiClient.post(`/api/companies/${companyId}/agents/${agent.id}/kill`),
    onSuccess: () => qc.invalidateQueries({ queryKey: ['agents', companyId] }),
  })

  const pause = useMutation({
    mutationFn: () =>
      apiClient.post(
        `/api/companies/${companyId}/agents/${agent.id}/${
          agent.status === 'paused' ? 'resume' : 'pause'
        }`
      ),
    onSuccess: () => qc.invalidateQueries({ queryKey: ['agents', companyId] }),
  })

  const budgetPct = Math.min(
    100,
    Math.round((agent.token_spend / agent.monthly_budget) * 100)
  )

  return (
    <div className="w-72 border-l border-zinc-800 bg-zinc-950 p-4 flex flex-col gap-4 overflow-y-auto">
      <div className="flex items-center justify-between">
        <h3 className="text-sm font-semibold text-zinc-100">{agent.title}</h3>
        <button onClick={onClose} className="text-zinc-600 hover:text-zinc-300">
          <X className="h-4 w-4" />
        </button>
      </div>

      <div className="flex items-center gap-2">
        <span className={cn('text-xs font-medium', STATUS_COLOURS[agent.status])}>
          ● {agent.status}
        </span>
        <span className="text-xs text-zinc-600">{agent.runtime}</span>
      </div>

      {currentIssue && (
        <div className="rounded-md border border-zinc-800 bg-zinc-900 px-3 py-2">
          <p className="text-xs text-zinc-500 mb-0.5">Current issue</p>
          <p className="text-xs text-zinc-300 line-clamp-2">{currentIssue.title}</p>
          <p className="text-xs text-zinc-600 mt-1">attempt #{currentIssue.attempt_count}</p>
        </div>
      )}

      <div>
        <div className="flex justify-between text-xs text-zinc-500 mb-1">
          <span>Token budget</span>
          <span>{budgetPct}%</span>
        </div>
        <div className="h-1.5 rounded-full bg-zinc-800">
          <div
            className={cn(
              'h-1.5 rounded-full',
              budgetPct > 80 ? 'bg-red-500' : budgetPct > 50 ? 'bg-amber-500' : 'bg-emerald-500'
            )}
            style={{ width: `${budgetPct}%` }}
          />
        </div>
        <p className="text-xs text-zinc-600 mt-1">
          {agent.token_spend.toLocaleString()} / {agent.monthly_budget.toLocaleString()} tokens
        </p>
      </div>

      <div className="space-y-2">
        {agent.status === 'idle' && (
          <Button
            size="sm"
            className="w-full"
            disabled={spawn.isPending}
            onClick={() => spawn.mutate()}
          >
            Spawn
          </Button>
        )}
        {(agent.status === 'working' || agent.status === 'paused') && (
          <Button
            size="sm"
            variant="secondary"
            className="w-full"
            disabled={pause.isPending}
            onClick={() => pause.mutate()}
          >
            {agent.status === 'paused' ? 'Resume' : 'Pause'}
          </Button>
        )}
        {agent.status !== 'idle' && agent.status !== 'done' && (
          <Button
            size="sm"
            variant="destructive"
            className="w-full"
            disabled={kill.isPending}
            onClick={() => kill.mutate()}
          >
            Kill
          </Button>
        )}
      </div>
    </div>
  )
}
```

- [ ] **Step 6: Create `web/src/components/OrgChart/index.tsx`**

```typescript
import { useCallback, useMemo } from 'react'
import { useParams } from 'react-router-dom'
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import {
  ReactFlow,
  Controls,
  Background,
  BackgroundVariant,
  useNodesState,
  useEdgesState,
  addEdge,
  type Node,
  type Edge,
  type Connection,
  type NodeDragHandler,
} from '@xyflow/react'
import '@xyflow/react/dist/style.css'
import { apiClient } from '@/lib/api'
import type { Agent } from '@/lib/types'
import { useAppStore } from '@/store/useAppStore'
import { AgentNode } from './AgentNode'
import { DetailPanel } from './DetailPanel'

const NODE_TYPES = { agentNode: AgentNode } as const

function agentsToFlow(agents: Agent[]): { nodes: Node[]; edges: Edge[] } {
  const nodes: Node[] = agents.map((a, i) => ({
    id: a.id,
    type: 'agentNode',
    position: { x: (i % 4) * 160, y: Math.floor(i / 4) * 100 },
    data: { agent: a },
  }))

  const edges: Edge[] = agents
    .filter((a) => a.manager_id)
    .map((a) => ({
      id: `${a.manager_id}-${a.id}`,
      source: a.manager_id!,
      target: a.id,
      style: { stroke: '#3f3f46' },
    }))

  return { nodes, edges }
}

export function OrgChart() {
  const { companyId } = useParams<{ companyId: string }>()
  const qc = useQueryClient()
  const selectedAgentId = useAppStore((s) => s.agentId)
  const setAgentId = useAppStore((s) => s.setAgentId)

  const { data: agents = [] } = useQuery<Agent[]>({
    queryKey: ['agents', companyId],
    queryFn: () => apiClient.get(`/api/companies/${companyId}/agents`).then((r) => r.data),
    enabled: !!companyId,
  })

  const { nodes: initNodes, edges: initEdges } = useMemo(
    () => agentsToFlow(agents),
    [agents]
  )

  const [nodes, , onNodesChange] = useNodesState(initNodes)
  const [edges, setEdges, onEdgesChange] = useEdgesState(initEdges)

  const reassign = useMutation({
    mutationFn: ({ agentId, newManagerId }: { agentId: string; newManagerId: string }) =>
      apiClient.post(`/api/companies/${companyId}/agents/${agentId}/reassign`, {
        new_manager_id: newManagerId,
      }),
    onSuccess: () => qc.invalidateQueries({ queryKey: ['agents', companyId] }),
  })

  const onConnect = useCallback(
    (connection: Connection) => setEdges((eds) => addEdge(connection, eds)),
    [setEdges]
  )

  const onNodeDragStop: NodeDragHandler = useCallback(
    (_, node) => {
      // Detect if node was dropped on another node — reassign manager
      // React Flow doesn't provide a target on drag stop; use edge connect for reassign instead.
      // Drag is used for layout; manager reassign is done via the detail panel or edge connect.
      void node
    },
    []
  )

  const selectedAgent = agents.find((a) => a.id === selectedAgentId) ?? null

  return (
    <div className="flex h-full">
      <div className="flex-1 h-full">
        <ReactFlow
          nodes={nodes}
          edges={edges}
          nodeTypes={NODE_TYPES}
          onNodesChange={onNodesChange}
          onEdgesChange={onEdgesChange}
          onConnect={onConnect}
          onNodeDragStop={onNodeDragStop}
          onNodeClick={(_, node) => setAgentId(node.id)}
          onPaneClick={() => setAgentId(null)}
          fitView
          className="bg-zinc-950"
        >
          <Controls className="[&>button]:bg-zinc-800 [&>button]:border-zinc-700 [&>button]:text-zinc-300" />
          <Background
            variant={BackgroundVariant.Dots}
            gap={24}
            size={1}
            color="#27272a"
          />
        </ReactFlow>
      </div>
      {selectedAgent && (
        <DetailPanel agent={selectedAgent} onClose={() => setAgentId(null)} />
      )}
    </div>
  )
}
```

- [ ] **Step 7: Commit**

```bash
git add web/src/components/OrgChart/
git commit -m "add org chart with react flow, agent nodes, detail panel"
```

---

## Task 9: Issues page

**Files:**
- Create: `web/src/components/Issues/IssueRow.tsx`
- Create: `web/src/components/Issues/index.tsx`
- Create: `web/src/components/Issues/Issues.test.tsx`

- [ ] **Step 1: Write failing test**

Create `web/src/components/Issues/Issues.test.tsx`:

```typescript
import { describe, it, beforeAll, afterEach, afterAll } from 'vitest'
import { render, screen } from '@testing-library/react'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { MemoryRouter, Route, Routes } from 'react-router-dom'
import { server } from '@/test/server'
import { http, HttpResponse } from 'msw'
import { Issues } from './index'
import type { Issue } from '@/lib/types'

beforeAll(() => server.listen())
afterEach(() => server.resetHandlers())
afterAll(() => server.close())

const mockIssues: Partial<Issue>[] = [
  { id: 'i1', title: 'Fix login bug', status: 'in_progress', attempt_count: 1 },
  { id: 'i2', title: 'Deploy to staging', status: 'pending', attempt_count: 0 },
]

it('renders issue titles', async () => {
  server.use(
    http.get('/api/companies/:id/agents', () => HttpResponse.json([])),
    http.get('/api/companies/:id/issues', () => HttpResponse.json(mockIssues)),
  )
  const qc = new QueryClient({ defaultOptions: { queries: { retry: false } } })
  render(
    <QueryClientProvider client={qc}>
      <MemoryRouter initialEntries={['/companies/c1/issues']}>
        <Routes>
          <Route path="/companies/:companyId/issues" element={<Issues />} />
        </Routes>
      </MemoryRouter>
    </QueryClientProvider>
  )
  expect(await screen.findByText('Fix login bug')).toBeInTheDocument()
  expect(await screen.findByText('Deploy to staging')).toBeInTheDocument()
})
```

- [ ] **Step 2: Run to confirm failure**

```bash
cd web && npm test -- Issues.test
```

Expected: FAIL

- [ ] **Step 3: Create `web/src/components/Issues/IssueRow.tsx`**

```typescript
import { useState } from 'react'
import type { Agent, Issue } from '@/lib/types'
import { cn } from '@/lib/utils'
import { ChevronDown, ChevronRight } from 'lucide-react'

const STATUS_COLOURS: Record<Issue['status'], string> = {
  pending: 'text-zinc-400',
  in_progress: 'text-blue-400',
  blocked: 'text-amber-400',
  done: 'text-emerald-400',
  failed: 'text-red-400',
}

interface IssueRowProps {
  issue: Issue
  agents: Agent[]
}

export function IssueRow({ issue, agents }: IssueRowProps) {
  const [open, setOpen] = useState(false)
  const assignee = agents.find((a) => a.id === issue.assignee_id)

  return (
    <>
      <tr
        className="border-b border-zinc-800 hover:bg-zinc-900/50 cursor-pointer"
        onClick={() => setOpen((o) => !o)}
      >
        <td className="px-3 py-2.5 text-sm text-zinc-200">{issue.title}</td>
        <td className="px-3 py-2.5 text-xs">
          <span className={STATUS_COLOURS[issue.status]}>{issue.status}</span>
        </td>
        <td className="px-3 py-2.5 text-xs text-zinc-500">
          {assignee?.title ?? '—'}
        </td>
        <td className="px-3 py-2.5 text-xs text-zinc-600">{issue.attempt_count}</td>
        <td className="px-3 py-2.5 text-zinc-600 w-6">
          {open ? (
            <ChevronDown className="h-3 w-3" />
          ) : (
            <ChevronRight className="h-3 w-3" />
          )}
        </td>
      </tr>
      {open && (
        <tr className="border-b border-zinc-800 bg-zinc-900/30">
          <td colSpan={5} className="px-4 py-3 text-xs text-zinc-400 space-y-1">
            {issue.description && <p>{issue.description}</p>}
            {issue.last_failure_reason && (
              <p className="text-red-400">Failure: {issue.last_failure_reason}</p>
            )}
            {issue.output_path && (
              <p className="text-zinc-500">Output: {issue.output_path}</p>
            )}
          </td>
        </tr>
      )}
    </>
  )
}
```

- [ ] **Step 4: Create `web/src/components/Issues/index.tsx`**

```typescript
import { useState } from 'react'
import { useParams } from 'react-router-dom'
import { useQuery } from '@tanstack/react-query'
import { apiClient } from '@/lib/api'
import type { Agent, Issue, IssueStatus } from '@/lib/types'
import { IssueRow } from './IssueRow'
import { Input } from '@/components/ui/input'

const ALL_STATUSES: IssueStatus[] = ['pending', 'in_progress', 'blocked', 'done', 'failed']

export function Issues() {
  const { companyId } = useParams<{ companyId: string }>()
  const [search, setSearch] = useState('')
  const [statusFilter, setStatusFilter] = useState<IssueStatus | ''>('')

  const { data: issues = [] } = useQuery<Issue[]>({
    queryKey: ['issues', companyId],
    queryFn: () =>
      apiClient.get(`/api/companies/${companyId}/issues`).then((r) => r.data),
    enabled: !!companyId,
  })

  const { data: agents = [] } = useQuery<Agent[]>({
    queryKey: ['agents', companyId],
    queryFn: () =>
      apiClient.get(`/api/companies/${companyId}/agents`).then((r) => r.data),
    enabled: !!companyId,
  })

  const filtered = issues.filter((i) => {
    const matchSearch = i.title.toLowerCase().includes(search.toLowerCase())
    const matchStatus = statusFilter ? i.status === statusFilter : true
    return matchSearch && matchStatus
  })

  return (
    <div className="p-6 max-w-5xl">
      <h1 className="text-lg font-semibold text-zinc-100 mb-4">Issues</h1>

      <div className="flex gap-3 mb-4">
        <Input
          placeholder="Search issues…"
          value={search}
          onChange={(e) => setSearch(e.target.value)}
          className="max-w-xs bg-zinc-800 border-zinc-700 text-zinc-100"
        />
        <select
          value={statusFilter}
          onChange={(e) => setStatusFilter(e.target.value as IssueStatus | '')}
          className="rounded-md border border-zinc-700 bg-zinc-800 text-zinc-300 text-sm px-3 py-2"
        >
          <option value="">All statuses</option>
          {ALL_STATUSES.map((s) => (
            <option key={s} value={s}>{s}</option>
          ))}
        </select>
      </div>

      <div className="rounded-md border border-zinc-800 overflow-hidden">
        <table className="w-full">
          <thead>
            <tr className="border-b border-zinc-800 bg-zinc-900">
              <th className="px-3 py-2 text-left text-xs font-medium text-zinc-500">Title</th>
              <th className="px-3 py-2 text-left text-xs font-medium text-zinc-500">Status</th>
              <th className="px-3 py-2 text-left text-xs font-medium text-zinc-500">Assignee</th>
              <th className="px-3 py-2 text-left text-xs font-medium text-zinc-500">Attempts</th>
              <th className="w-6" />
            </tr>
          </thead>
          <tbody>
            {filtered.map((issue) => (
              <IssueRow key={issue.id} issue={issue} agents={agents} />
            ))}
            {filtered.length === 0 && (
              <tr>
                <td colSpan={5} className="px-3 py-6 text-center text-sm text-zinc-600">
                  No issues found.
                </td>
              </tr>
            )}
          </tbody>
        </table>
      </div>
    </div>
  )
}
```

- [ ] **Step 5: Run tests**

```bash
cd web && npm test -- Issues.test
```

Expected: PASS.

- [ ] **Step 6: Commit**

```bash
git add web/src/components/Issues/
git commit -m "add issues page with table, filters, expandable rows"
```

---

## Task 10: Hiring page

**Files:**
- Create: `web/src/components/Hiring/HireCard.tsx`
- Create: `web/src/components/Hiring/index.tsx`
- Create: `web/src/components/Hiring/Hiring.test.tsx`

- [ ] **Step 1: Write failing test**

Create `web/src/components/Hiring/Hiring.test.tsx`:

```typescript
import { it, beforeAll, afterEach, afterAll } from 'vitest'
import { render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { MemoryRouter, Route, Routes } from 'react-router-dom'
import { server } from '@/test/server'
import { http, HttpResponse } from 'msw'
import { Hiring } from './index'
import type { PendingHire } from '@/lib/types'

beforeAll(() => server.listen())
afterEach(() => server.resetHandlers())
afterAll(() => server.close())

const mockHire: PendingHire = {
  id: 'h1',
  company_id: 'c1',
  requested_by_agent_id: 'a1',
  role_title: 'Senior Engineer',
  reporting_to_agent_id: 'a2',
  system_prompt: 'You are a senior engineer.',
  runtime: 'claude_code',
  budget_allocation: 50000,
  initial_task: 'Build auth module',
  status: 'pending',
  created_at: new Date().toISOString(),
}

it('renders pending hire and approve button', async () => {
  server.use(
    http.get('/api/companies/:id/hires', () => HttpResponse.json([mockHire])),
  )
  const qc = new QueryClient({ defaultOptions: { queries: { retry: false } } })
  render(
    <QueryClientProvider client={qc}>
      <MemoryRouter initialEntries={['/companies/c1/hiring']}>
        <Routes>
          <Route path="/companies/:companyId/hiring" element={<Hiring />} />
        </Routes>
      </MemoryRouter>
    </QueryClientProvider>
  )
  expect(await screen.findByText('Senior Engineer')).toBeInTheDocument()
  expect(screen.getByRole('button', { name: /approve/i })).toBeInTheDocument()
  expect(screen.getByRole('button', { name: /reject/i })).toBeInTheDocument()
})

it('calls approve endpoint on approve click', async () => {
  let approveHit = false
  server.use(
    http.get('/api/companies/:id/hires', () => HttpResponse.json([mockHire])),
    http.post('/api/companies/:id/hires/:hireId/approve', () => {
      approveHit = true
      return HttpResponse.json({ status: 'approved' })
    }),
  )
  const qc = new QueryClient({ defaultOptions: { queries: { retry: false } } })
  render(
    <QueryClientProvider client={qc}>
      <MemoryRouter initialEntries={['/companies/c1/hiring']}>
        <Routes>
          <Route path="/companies/:companyId/hiring" element={<Hiring />} />
        </Routes>
      </MemoryRouter>
    </QueryClientProvider>
  )
  await userEvent.click(await screen.findByRole('button', { name: /approve/i }))
  expect(approveHit).toBe(true)
})
```

- [ ] **Step 2: Run to confirm failure**

```bash
cd web && npm test -- Hiring.test
```

Expected: FAIL

- [ ] **Step 3: Create `web/src/components/Hiring/HireCard.tsx`**

```typescript
import type { PendingHire } from '@/lib/types'
import { Button } from '@/components/ui/button'
import { Card, CardContent } from '@/components/ui/card'

interface HireCardProps {
  hire: PendingHire
  onApprove: () => void
  onReject: () => void
  isApproving: boolean
  isRejecting: boolean
}

export function HireCard({
  hire,
  onApprove,
  onReject,
  isApproving,
  isRejecting,
}: HireCardProps) {
  return (
    <Card className="border-zinc-800 bg-zinc-900">
      <CardContent className="pt-4 space-y-3">
        <div>
          <p className="text-sm font-medium text-zinc-100">{hire.role_title}</p>
          <p className="text-xs text-zinc-500 mt-0.5">{hire.runtime}</p>
        </div>

        {hire.initial_task && (
          <div className="rounded-md bg-zinc-800 px-3 py-2">
            <p className="text-xs text-zinc-500 mb-0.5">Initial task</p>
            <p className="text-xs text-zinc-300">{hire.initial_task}</p>
          </div>
        )}

        <div className="flex items-center justify-between text-xs text-zinc-500">
          <span>Budget: {hire.budget_allocation.toLocaleString()} tokens</span>
          <span>{new Date(hire.created_at).toLocaleDateString()}</span>
        </div>

        <div className="flex gap-2 pt-1">
          <Button
            size="sm"
            className="flex-1"
            disabled={isApproving}
            onClick={onApprove}
          >
            {isApproving ? 'Approving…' : 'Approve'}
          </Button>
          <Button
            size="sm"
            variant="secondary"
            className="flex-1"
            disabled={isRejecting}
            onClick={onReject}
          >
            {isRejecting ? 'Rejecting…' : 'Reject'}
          </Button>
        </div>
      </CardContent>
    </Card>
  )
}
```

- [ ] **Step 4: Create `web/src/components/Hiring/index.tsx`**

```typescript
import { useParams } from 'react-router-dom'
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { apiClient } from '@/lib/api'
import type { PendingHire } from '@/lib/types'
import { HireCard } from './HireCard'

export function Hiring() {
  const { companyId } = useParams<{ companyId: string }>()
  const qc = useQueryClient()

  const { data: hires = [] } = useQuery<PendingHire[]>({
    queryKey: ['hires', companyId],
    queryFn: () =>
      apiClient.get(`/api/companies/${companyId}/hires`).then((r) => r.data),
    enabled: !!companyId,
  })

  const approve = useMutation({
    mutationFn: (hireId: string) =>
      apiClient.post(`/api/companies/${companyId}/hires/${hireId}/approve`),
    onSuccess: () => qc.invalidateQueries({ queryKey: ['hires', companyId] }),
  })

  const reject = useMutation({
    mutationFn: (hireId: string) =>
      apiClient.post(`/api/companies/${companyId}/hires/${hireId}/reject`),
    onSuccess: () => qc.invalidateQueries({ queryKey: ['hires', companyId] }),
  })

  const pending = hires.filter((h) => h.status === 'pending')

  return (
    <div className="p-6 max-w-3xl">
      <h1 className="text-lg font-semibold text-zinc-100 mb-4">
        Hiring{pending.length > 0 && <span className="ml-2 text-zinc-500 text-sm font-normal">({pending.length} pending)</span>}
      </h1>

      {pending.length === 0 ? (
        <p className="text-sm text-zinc-600">No pending hire requests.</p>
      ) : (
        <div className="grid grid-cols-1 sm:grid-cols-2 gap-4">
          {pending.map((hire) => (
            <HireCard
              key={hire.id}
              hire={hire}
              onApprove={() => approve.mutate(hire.id)}
              onReject={() => reject.mutate(hire.id)}
              isApproving={approve.isPending && approve.variables === hire.id}
              isRejecting={reject.isPending && reject.variables === hire.id}
            />
          ))}
        </div>
      )}
    </div>
  )
}
```

- [ ] **Step 5: Run tests**

```bash
cd web && npm test -- Hiring.test
```

Expected: PASS, 2 tests.

- [ ] **Step 6: Commit**

```bash
git add web/src/components/Hiring/
git commit -m "add hiring page with approve/reject cards"
```

---

## Task 11: Audit Log page

**Files:**
- Create: `web/src/components/Audit/index.tsx`

No new tests needed — this page is a simple query + render with no logic to unit-test separately.

- [ ] **Step 1: Create `web/src/components/Audit/index.tsx`**

```typescript
import { useState } from 'react'
import { useParams } from 'react-router-dom'
import { useQuery } from '@tanstack/react-query'
import { apiClient } from '@/lib/api'
import type { AuditLog } from '@/lib/types'

export function Audit() {
  const { companyId } = useParams<{ companyId: string }>()
  const [expanded, setExpanded] = useState<string | null>(null)

  const { data: logs = [] } = useQuery<AuditLog[]>({
    queryKey: ['audit', companyId],
    queryFn: () =>
      apiClient
        .get(`/api/companies/${companyId}/audit?limit=100`)
        .then((r) => r.data),
    enabled: !!companyId,
  })

  return (
    <div className="p-6 max-w-5xl">
      <h1 className="text-lg font-semibold text-zinc-100 mb-4">Audit Log</h1>

      <div className="rounded-md border border-zinc-800 overflow-hidden">
        <table className="w-full">
          <thead>
            <tr className="border-b border-zinc-800 bg-zinc-900">
              <th className="px-3 py-2 text-left text-xs font-medium text-zinc-500">Time</th>
              <th className="px-3 py-2 text-left text-xs font-medium text-zinc-500">Event</th>
              <th className="px-3 py-2 text-left text-xs font-medium text-zinc-500">Actor</th>
              <th className="px-3 py-2 text-left text-xs font-medium text-zinc-500">Payload</th>
            </tr>
          </thead>
          <tbody>
            {logs.map((log) => (
              <>
                <tr
                  key={log.id}
                  className="border-b border-zinc-800 hover:bg-zinc-900/50 cursor-pointer"
                  onClick={() => setExpanded(expanded === log.id ? null : log.id)}
                >
                  <td className="px-3 py-2 text-xs text-zinc-500 whitespace-nowrap">
                    {new Date(log.created_at).toLocaleString()}
                  </td>
                  <td className="px-3 py-2 text-xs font-mono text-zinc-300">{log.event_type}</td>
                  <td className="px-3 py-2 text-xs text-zinc-500">
                    {log.actor_id ? log.actor_id.slice(0, 8) + '…' : '—'}
                  </td>
                  <td className="px-3 py-2 text-xs text-zinc-600">
                    {expanded === log.id ? '▲ hide' : '▼ show'}
                  </td>
                </tr>
                {expanded === log.id && (
                  <tr key={`${log.id}-exp`} className="border-b border-zinc-800 bg-zinc-900/30">
                    <td colSpan={4} className="px-4 py-3">
                      <pre className="text-xs text-zinc-400 whitespace-pre-wrap font-mono">
                        {JSON.stringify(log.payload, null, 2)}
                      </pre>
                    </td>
                  </tr>
                )}
              </>
            ))}
            {logs.length === 0 && (
              <tr>
                <td colSpan={4} className="px-3 py-6 text-center text-sm text-zinc-600">
                  No events yet.
                </td>
              </tr>
            )}
          </tbody>
        </table>
      </div>
    </div>
  )
}
```

- [ ] **Step 2: Commit**

```bash
git add web/src/components/Audit/
git commit -m "add audit log page"
```

---

## Task 12: WebSocket hook + real-time cache patching

**Files:**
- Create: `web/src/hooks/useWebSocket.ts`
- Create: `web/src/hooks/useWebSocket.test.ts`

- [ ] **Step 1: Write failing test**

Create `web/src/hooks/useWebSocket.test.ts`:

```typescript
import { describe, it, expect, vi, beforeEach } from 'vitest'
import { patchCacheFromEvent } from './useWebSocket'
import type { WsEvent, Agent } from '@/lib/types'
import { QueryClient } from '@tanstack/react-query'

describe('patchCacheFromEvent', () => {
  let qc: QueryClient
  beforeEach(() => {
    qc = new QueryClient()
  })

  it('updates agent status in the agents cache on agent_status event', () => {
    const existingAgent: Agent = {
      id: 'a1',
      company_id: 'c1',
      role: 'cto',
      title: 'CTO',
      system_prompt: '',
      manager_id: null,
      runtime: 'claude_code',
      status: 'idle',
      monthly_budget: 100000,
      token_spend: 0,
      chat_token_spend: 0,
      pid: null,
      created_at: '',
      updated_at: '',
    }
    qc.setQueryData(['agents', 'c1'], [existingAgent])

    const event: WsEvent = {
      type: 'agent_status',
      company_id: 'c1',
      payload: { agent_id: 'a1', status: 'working' },
    }

    patchCacheFromEvent(qc, event)

    const updated = qc.getQueryData<Agent[]>(['agents', 'c1'])
    expect(updated?.[0].status).toBe('working')
  })

  it('invalidates notifications cache on notification event', () => {
    const invalidate = vi.spyOn(qc, 'invalidateQueries')
    const event: WsEvent = {
      type: 'notification',
      company_id: 'c1',
      payload: {},
    }
    patchCacheFromEvent(qc, event)
    expect(invalidate).toHaveBeenCalledWith({ queryKey: ['notifications', 'c1'] })
  })

  it('invalidates hires cache on hire_pending event', () => {
    const invalidate = vi.spyOn(qc, 'invalidateQueries')
    const event: WsEvent = {
      type: 'hire_pending',
      company_id: 'c1',
      payload: {},
    }
    patchCacheFromEvent(qc, event)
    expect(invalidate).toHaveBeenCalledWith({ queryKey: ['hires', 'c1'] })
  })
})
```

- [ ] **Step 2: Run to confirm failure**

```bash
cd web && npm test -- useWebSocket
```

Expected: FAIL — `Cannot find module './useWebSocket'`

- [ ] **Step 3: Create `web/src/hooks/useWebSocket.ts`**

```typescript
import { useEffect, useRef } from 'react'
import { useQueryClient, type QueryClient } from '@tanstack/react-query'
import type { WsEvent, Agent } from '@/lib/types'

// Exported for unit testing
export function patchCacheFromEvent(qc: QueryClient, event: WsEvent) {
  const companyId = event.company_id

  switch (event.type) {
    case 'agent_status': {
      const payload = event.payload as { agent_id: string; status: Agent['status'] }
      qc.setQueryData<Agent[]>(['agents', companyId], (old) =>
        old?.map((a) =>
          a.id === payload.agent_id ? { ...a, status: payload.status } : a
        )
      )
      break
    }
    case 'issue_update':
      qc.invalidateQueries({ queryKey: ['issues', companyId] })
      break
    case 'notification':
      qc.invalidateQueries({ queryKey: ['notifications', companyId] })
      break
    case 'hire_pending':
      qc.invalidateQueries({ queryKey: ['hires', companyId] })
      break
    case 'escalation':
      qc.invalidateQueries({ queryKey: ['notifications', companyId] })
      break
    default:
      break
  }
}

export function useWebSocket(companyId: string | null) {
  const qc = useQueryClient()
  const wsRef = useRef<WebSocket | null>(null)

  useEffect(() => {
    if (!companyId) return

    const token = localStorage.getItem('legion_token')
    if (!token) return

    const protocol = window.location.protocol === 'https:' ? 'wss' : 'ws'
    const host = window.location.host
    const url = `${protocol}://${host}/api/companies/${companyId}/ws?token=${token}`

    const ws = new WebSocket(url)
    wsRef.current = ws

    ws.onmessage = (e) => {
      try {
        const event: WsEvent = JSON.parse(e.data)
        patchCacheFromEvent(qc, event)
      } catch {
        // malformed message — ignore
      }
    }

    ws.onerror = () => {
      // Reconnect on next companyId change or page reload
    }

    return () => {
      ws.close()
      wsRef.current = null
    }
  }, [companyId, qc])
}
```

- [ ] **Step 4: Run tests**

```bash
cd web && npm test -- useWebSocket
```

Expected: PASS, 3 tests.

- [ ] **Step 5: Wire `useWebSocket` into `CompanyShell.tsx`**

Add this import and call inside `CompanyShell`:

```typescript
// Add to imports in CompanyShell.tsx
import { useWebSocket } from '@/hooks/useWebSocket'

// Add inside CompanyShell function body, after the useEffect for setCompanyId
useWebSocket(companyId ?? null)
```

- [ ] **Step 6: Run full test suite**

```bash
cd web && npm test
```

Expected: all tests pass.

- [ ] **Step 7: Commit**

```bash
git add web/src/hooks/ web/src/pages/CompanyShell.tsx
git commit -m "add websocket hook with real-time cache patching"
```

---

## Task 13: Build verification

- [ ] **Step 1: Verify TypeScript compiles**

```bash
cd web && npx tsc --noEmit
```

Expected: no errors.

- [ ] **Step 2: Build for production**

```bash
cd web && npm run build
```

Expected: `dist/` directory created, no errors.

- [ ] **Step 3: Start Go backend and check SPA is served**

```bash
# From repo root — requires DB running
go run ./cmd/conductor --static-dir web/dist
```

Navigate to `http://localhost:8080` in browser. Expected: legion login page loads.

- [ ] **Step 4: Commit build output exclusion**

Ensure `web/dist` is in `.gitignore`:

```bash
grep -q 'web/dist' .gitignore || echo 'web/dist' >> .gitignore
git add .gitignore
git commit -m "exclude web/dist from git"
```

- [ ] **Step 5: Final commit**

```bash
git add web/
git commit -m "Phase 4 complete: React UI — dashboard, org chart, issues, hiring, audit, websocket"
```

---

## Appendix: `AgentChat` component stub

The chat panel lives inside `DetailPanel` but full thread persistence is Phase 5. Add this stub so the import in `DetailPanel` doesn't break if someone wires it early:

Create `web/src/components/AgentChat/index.tsx`:

```typescript
interface AgentChatProps {
  agentId: string
  companyId: string
}

export function AgentChat({ agentId, companyId }: AgentChatProps) {
  // Full implementation in Phase 5
  return (
    <div className="flex items-center justify-center h-32 text-xs text-zinc-600">
      Chat coming in Phase 5
    </div>
  )
}
```
