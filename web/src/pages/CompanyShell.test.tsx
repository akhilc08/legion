import { describe, it, expect, vi, beforeAll, afterEach, afterAll, beforeEach } from 'vitest'
import { render, screen, waitFor } from '@testing-library/react'
import { MemoryRouter, Route, Routes } from 'react-router-dom'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { http, HttpResponse } from 'msw'
import { server } from '@/test/server'
import type { Agent, PendingHire } from '@/lib/types'

// ---- Mock heavy child components ----
vi.mock('@/components/Layout/Sidebar', () => ({
  Sidebar: ({ hireBadgeCount, agents }: { hireBadgeCount: number; agents: Agent[] }) => (
    <div data-testid="sidebar" data-badge={hireBadgeCount} data-agents={agents.length} />
  ),
}))
vi.mock('@/components/Dashboard', () => ({
  Dashboard: () => <div data-testid="dashboard" />,
}))
vi.mock('@/components/OrgChart', () => ({
  OrgChart: () => <div data-testid="org-chart" />,
}))
vi.mock('@/components/Issues', () => ({
  Issues: () => <div data-testid="issues" />,
}))
vi.mock('@/components/Hiring', () => ({
  Hiring: () => <div data-testid="hiring" />,
}))
vi.mock('@/components/Audit', () => ({
  Audit: () => <div data-testid="audit" />,
}))
vi.mock('@/components/FSBrowser', () => ({
  FSBrowser: () => <div data-testid="fsbrowser" />,
}))
vi.mock('@/components/OrgChart/DetailPanel', () => ({
  DetailPanel: ({ agent, onClose }: { agent: Agent; onClose: () => void }) => (
    <div data-testid="detail-panel" data-agent-id={agent.id}>
      <button onClick={onClose}>Close</button>
    </div>
  ),
}))

// ---- Mock useWebSocket ----
vi.mock('@/hooks/useWebSocket', () => ({
  useWebSocket: vi.fn(),
}))

// ---- Store mock ----
const mockSetCompanyId = vi.fn()
const mockSetAgentId = vi.fn()
let mockAgentId: string | null = null

vi.mock('@/store/useAppStore', () => ({
  useAppStore: (sel: (s: {
    setCompanyId: (id: string | null) => void
    agentId: string | null
    setAgentId: (id: string | null) => void
  }) => unknown) =>
    sel({
      setCompanyId: mockSetCompanyId,
      agentId: mockAgentId,
      setAgentId: mockSetAgentId,
    }),
}))

import { CompanyShell } from './CompanyShell'

const COMPANY_ID = 'test-company-id'

const AGENTS: Agent[] = [
  {
    id: 'agent-1',
    company_id: COMPANY_ID,
    role: 'cto',
    title: 'CTO Agent',
    system_prompt: '',
    manager_id: null,
    runtime: 'claude_code',
    status: 'idle',
    monthly_budget: 100000,
    token_spend: 0,
    chat_token_spend: 0,
    pid: null,
    created_at: '2024-01-01T00:00:00Z',
    updated_at: '2024-01-01T00:00:00Z',
  },
]

const HIRES: PendingHire[] = [
  {
    id: 'hire-1',
    company_id: COMPANY_ID,
    requested_by_agent_id: 'agent-1',
    role_title: 'Engineer',
    reporting_to_agent_id: 'agent-1',
    system_prompt: '',
    runtime: 'claude_code',
    budget_allocation: 1000,
    initial_task: null,
    status: 'pending',
    created_at: '2024-01-01T00:00:00Z',
  },
  {
    id: 'hire-2',
    company_id: COMPANY_ID,
    requested_by_agent_id: 'agent-1',
    role_title: 'Designer',
    reporting_to_agent_id: 'agent-1',
    system_prompt: '',
    runtime: 'claude_code',
    budget_allocation: 1000,
    initial_task: null,
    status: 'approved',
    created_at: '2024-01-01T00:00:00Z',
  },
]

function makeQueryClient() {
  return new QueryClient({ defaultOptions: { queries: { retry: false } } })
}

function renderShell(
  path = `/companies/${COMPANY_ID}/dashboard`,
  qc?: QueryClient
) {
  const client = qc ?? makeQueryClient()
  return render(
    <QueryClientProvider client={client}>
      <MemoryRouter initialEntries={[path]}>
        <Routes>
          <Route path="/companies/:companyId/*" element={<CompanyShell />} />
        </Routes>
      </MemoryRouter>
    </QueryClientProvider>
  )
}

beforeAll(() => server.listen({ onUnhandledRequest: 'bypass' }))
afterEach(() => {
  server.resetHandlers()
  mockSetCompanyId.mockReset()
  mockSetAgentId.mockReset()
  mockAgentId = null
})
afterAll(() => server.close())

describe('CompanyShell – mounting', () => {
  beforeEach(() => {
    server.use(
      http.get(`/api/companies/${COMPANY_ID}/agents`, () => HttpResponse.json(AGENTS)),
      http.get(`/api/companies/${COMPANY_ID}/hires`, () => HttpResponse.json(HIRES))
    )
  })

  it('renders without crashing', () => {
    expect(() => renderShell()).not.toThrow()
  })

  it('renders Sidebar', async () => {
    renderShell()
    expect(await screen.findByTestId('sidebar')).toBeInTheDocument()
  })

  it('calls setCompanyId with the URL companyId param', async () => {
    renderShell()
    await waitFor(() => expect(mockSetCompanyId).toHaveBeenCalledWith(COMPANY_ID))
  })

  it('calls setCompanyId exactly once on mount', async () => {
    renderShell()
    await waitFor(() => expect(mockSetCompanyId).toHaveBeenCalledOnce())
  })
})

describe('CompanyShell – Sidebar hire badge count', () => {
  it('passes pendingCount (only "pending" hires) to Sidebar', async () => {
    server.use(
      http.get(`/api/companies/${COMPANY_ID}/agents`, () => HttpResponse.json(AGENTS)),
      http.get(`/api/companies/${COMPANY_ID}/hires`, () => HttpResponse.json(HIRES))
    )
    renderShell()
    const sidebar = await screen.findByTestId('sidebar')
    // HIRES has 1 pending + 1 approved, so badge should be 1
    await waitFor(() => expect(sidebar.dataset.badge).toBe('1'))
  })

  it('passes zero badge count when there are no pending hires', async () => {
    const noHires: PendingHire[] = []
    server.use(
      http.get(`/api/companies/${COMPANY_ID}/agents`, () => HttpResponse.json(AGENTS)),
      http.get(`/api/companies/${COMPANY_ID}/hires`, () => HttpResponse.json(noHires))
    )
    renderShell()
    const sidebar = await screen.findByTestId('sidebar')
    await waitFor(() => expect(sidebar.dataset.badge).toBe('0'))
  })

  it('passes correct agent count to Sidebar', async () => {
    server.use(
      http.get(`/api/companies/${COMPANY_ID}/agents`, () => HttpResponse.json(AGENTS)),
      http.get(`/api/companies/${COMPANY_ID}/hires`, () => HttpResponse.json([]))
    )
    renderShell()
    const sidebar = await screen.findByTestId('sidebar')
    await waitFor(() => expect(sidebar.dataset.agents).toBe('1'))
  })
})

describe('CompanyShell – DetailPanel', () => {
  beforeEach(() => {
    server.use(
      http.get(`/api/companies/${COMPANY_ID}/agents`, () => HttpResponse.json(AGENTS)),
      http.get(`/api/companies/${COMPANY_ID}/hires`, () => HttpResponse.json([]))
    )
  })

  it('does NOT render DetailPanel when agentId is null', async () => {
    mockAgentId = null
    renderShell()
    await screen.findByTestId('sidebar')
    expect(screen.queryByTestId('detail-panel')).not.toBeInTheDocument()
  })

  it('renders DetailPanel when a matching agentId is set', async () => {
    mockAgentId = 'agent-1'
    renderShell()
    await waitFor(() => expect(screen.queryByTestId('detail-panel')).toBeInTheDocument())
  })

  it('passes the correct agent to DetailPanel', async () => {
    mockAgentId = 'agent-1'
    renderShell()
    const panel = await screen.findByTestId('detail-panel')
    expect(panel.dataset.agentId).toBe('agent-1')
  })
})

describe('CompanyShell – route navigation', () => {
  beforeEach(() => {
    server.use(
      http.get(`/api/companies/${COMPANY_ID}/agents`, () => HttpResponse.json(AGENTS)),
      http.get(`/api/companies/${COMPANY_ID}/hires`, () => HttpResponse.json([]))
    )
  })

  it('renders Dashboard at /companies/:id/dashboard', async () => {
    renderShell(`/companies/${COMPANY_ID}/dashboard`)
    expect(await screen.findByTestId('dashboard')).toBeInTheDocument()
  })

  it('renders OrgChart at /companies/:id/org-chart', async () => {
    renderShell(`/companies/${COMPANY_ID}/org-chart`)
    expect(await screen.findByTestId('org-chart')).toBeInTheDocument()
  })

  it('renders Issues at /companies/:id/issues', async () => {
    renderShell(`/companies/${COMPANY_ID}/issues`)
    expect(await screen.findByTestId('issues')).toBeInTheDocument()
  })

  it('renders Hiring at /companies/:id/hiring', async () => {
    renderShell(`/companies/${COMPANY_ID}/hiring`)
    expect(await screen.findByTestId('hiring')).toBeInTheDocument()
  })

  it('renders Audit at /companies/:id/audit', async () => {
    renderShell(`/companies/${COMPANY_ID}/audit`)
    expect(await screen.findByTestId('audit')).toBeInTheDocument()
  })

  it('renders FSBrowser at /companies/:id/files', async () => {
    renderShell(`/companies/${COMPANY_ID}/files`)
    expect(await screen.findByTestId('fsbrowser')).toBeInTheDocument()
  })

  it('index route redirects to dashboard', async () => {
    renderShell(`/companies/${COMPANY_ID}`)
    // After redirect the dashboard should render
    expect(await screen.findByTestId('dashboard')).toBeInTheDocument()
  })
})

describe('CompanyShell – null/empty API responses', () => {
  it('handles null hires response gracefully (defaults to [])', async () => {
    server.use(
      http.get(`/api/companies/${COMPANY_ID}/agents`, () => HttpResponse.json(AGENTS)),
      http.get(`/api/companies/${COMPANY_ID}/hires`, () => HttpResponse.json(null))
    )
    renderShell()
    const sidebar = await screen.findByTestId('sidebar')
    await waitFor(() => expect(sidebar.dataset.badge).toBe('0'))
  })

  it('handles null agents response gracefully (defaults to [])', async () => {
    server.use(
      http.get(`/api/companies/${COMPANY_ID}/agents`, () => HttpResponse.json(null)),
      http.get(`/api/companies/${COMPANY_ID}/hires`, () => HttpResponse.json([]))
    )
    renderShell()
    const sidebar = await screen.findByTestId('sidebar')
    await waitFor(() => expect(sidebar.dataset.agents).toBe('0'))
  })
})
