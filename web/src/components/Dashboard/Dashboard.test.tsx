import { describe, it, expect, beforeAll, afterEach, afterAll } from 'vitest'
import { render, screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { MemoryRouter, Route, Routes } from 'react-router-dom'
import { server } from '@/test/server'
import { http, HttpResponse } from 'msw'
import { Dashboard } from './index'
import { AgentStatusCard } from './AgentStatusCard'
import { EscalationList } from './EscalationList'
import type { Agent, Notification } from '@/lib/types'

beforeAll(() => server.listen())
afterEach(() => server.resetHandlers())
afterAll(() => server.close())

function makeWrapper(companyId: string) {
  const qc = new QueryClient({ defaultOptions: { queries: { retry: false }, mutations: { retry: false } } })
  return ({ children }: { children: React.ReactNode }) => (
    <QueryClientProvider client={qc}>
      <MemoryRouter initialEntries={[`/companies/${companyId}/dashboard`]}>
        <Routes><Route path="/companies/:companyId/dashboard" element={<>{children}</>} /></Routes>
      </MemoryRouter>
    </QueryClientProvider>
  )
}

const makeAgent = (id: string, status: Agent['status']): Agent => ({
  id, company_id: 'c1', role: 'engineer', title: `Agent ${id}`,
  system_prompt: '', manager_id: null, runtime: 'claude_code',
  status, monthly_budget: 10000, token_spend: 0, chat_token_spend: 0,
  pid: null, created_at: new Date().toISOString(), updated_at: new Date().toISOString(),
})

const makeNotification = (id: string, type = 'escalation'): Notification => ({
  id, company_id: 'c1', type, escalation_id: null,
  payload: {}, dismissed_at: null, created_at: new Date().toISOString(),
})

// ── Heading ───────────────────────────────────────────────────────────────────

describe('Dashboard heading', () => {
  it('renders "Dashboard" heading', async () => {
    server.use(
      http.get('/api/companies/:id/agents', () => HttpResponse.json([])),
      http.get('/api/companies/:id/notifications', () => HttpResponse.json([])),
    )
    render(<Dashboard />, { wrapper: makeWrapper('c1') })
    expect(await screen.findByText('Dashboard')).toBeInTheDocument()
  })
})

// ── Status counts ─────────────────────────────────────────────────────────────

describe('Dashboard status counts', () => {
  it('shows working count correctly (2 working agents)', async () => {
    server.use(
      http.get('/api/companies/:id/agents', () => HttpResponse.json([
        makeAgent('1', 'working'), makeAgent('2', 'working'), makeAgent('3', 'idle'),
      ])),
      http.get('/api/companies/:id/notifications', () => HttpResponse.json([])),
    )
    render(<Dashboard />, { wrapper: makeWrapper('c1') })
    await waitFor(() => {
      const workingCard = screen.getByText('Working').closest('div')!
      expect(workingCard.querySelector('p')?.textContent).toBe('2')
    })
  })

  it('shows idle count correctly', async () => {
    server.use(
      http.get('/api/companies/:id/agents', () => HttpResponse.json([
        makeAgent('1', 'idle'), makeAgent('2', 'idle'), makeAgent('3', 'working'),
      ])),
      http.get('/api/companies/:id/notifications', () => HttpResponse.json([])),
    )
    render(<Dashboard />, { wrapper: makeWrapper('c1') })
    await waitFor(() => {
      const idleCard = screen.getByText('Idle').closest('div')!
      expect(idleCard.querySelector('p')?.textContent).toBe('2')
    })
  })

  it('shows blocked count correctly', async () => {
    server.use(
      http.get('/api/companies/:id/agents', () => HttpResponse.json([
        makeAgent('1', 'blocked'),
      ])),
      http.get('/api/companies/:id/notifications', () => HttpResponse.json([])),
    )
    render(<Dashboard />, { wrapper: makeWrapper('c1') })
    await waitFor(() => {
      const blockedCard = screen.getByText('Blocked').closest('div')!
      expect(blockedCard.querySelector('p')?.textContent).toBe('1')
    })
  })

  it('shows failed count correctly', async () => {
    server.use(
      http.get('/api/companies/:id/agents', () => HttpResponse.json([
        makeAgent('1', 'failed'), makeAgent('2', 'failed'), makeAgent('3', 'failed'),
      ])),
      http.get('/api/companies/:id/notifications', () => HttpResponse.json([])),
    )
    render(<Dashboard />, { wrapper: makeWrapper('c1') })
    await waitFor(() => {
      const failedCard = screen.getByText('Failed').closest('div')!
      expect(failedCard.querySelector('p')?.textContent).toBe('3')
    })
  })

  it('shows degraded count correctly', async () => {
    server.use(
      http.get('/api/companies/:id/agents', () => HttpResponse.json([
        makeAgent('1', 'degraded'),
      ])),
      http.get('/api/companies/:id/notifications', () => HttpResponse.json([])),
    )
    render(<Dashboard />, { wrapper: makeWrapper('c1') })
    await waitFor(() => {
      const degradedCard = screen.getByText('Degraded').closest('div')!
      expect(degradedCard.querySelector('p')?.textContent).toBe('1')
    })
  })

  it('all counts are 0 when no agents', async () => {
    server.use(
      http.get('/api/companies/:id/agents', () => HttpResponse.json([])),
      http.get('/api/companies/:id/notifications', () => HttpResponse.json([])),
    )
    render(<Dashboard />, { wrapper: makeWrapper('c1') })
    await screen.findByText('Dashboard')
    const zeroElements = screen.getAllByText('0')
    expect(zeroElements.length).toBeGreaterThanOrEqual(5)
  })
})

// ── Status card labels ────────────────────────────────────────────────────────

describe('Dashboard status card labels', () => {
  function setupEmpty() {
    server.use(
      http.get('/api/companies/:id/agents', () => HttpResponse.json([])),
      http.get('/api/companies/:id/notifications', () => HttpResponse.json([])),
    )
  }

  it('shows "Working" label', async () => {
    setupEmpty()
    render(<Dashboard />, { wrapper: makeWrapper('c1') })
    expect(await screen.findByText('Working')).toBeInTheDocument()
  })

  it('shows "Idle" label', async () => {
    setupEmpty()
    render(<Dashboard />, { wrapper: makeWrapper('c1') })
    expect(await screen.findByText('Idle')).toBeInTheDocument()
  })

  it('shows "Blocked" label', async () => {
    setupEmpty()
    render(<Dashboard />, { wrapper: makeWrapper('c1') })
    expect(await screen.findByText('Blocked')).toBeInTheDocument()
  })

  it('shows "Failed" label', async () => {
    setupEmpty()
    render(<Dashboard />, { wrapper: makeWrapper('c1') })
    expect(await screen.findByText('Failed')).toBeInTheDocument()
  })

  it('shows "Degraded" label', async () => {
    setupEmpty()
    render(<Dashboard />, { wrapper: makeWrapper('c1') })
    expect(await screen.findByText('Degraded')).toBeInTheDocument()
  })
})

// ── Notifications ─────────────────────────────────────────────────────────────

describe('Dashboard notifications', () => {
  it('Notifications section heading shown', async () => {
    server.use(
      http.get('/api/companies/:id/agents', () => HttpResponse.json([])),
      http.get('/api/companies/:id/notifications', () => HttpResponse.json([])),
    )
    render(<Dashboard />, { wrapper: makeWrapper('c1') })
    expect(await screen.findByText('Notifications')).toBeInTheDocument()
  })

  it('notification listed when present', async () => {
    server.use(
      http.get('/api/companies/:id/agents', () => HttpResponse.json([])),
      http.get('/api/companies/:id/notifications', () =>
        HttpResponse.json([makeNotification('n1', 'agent_blocked')]),
      ),
    )
    render(<Dashboard />, { wrapper: makeWrapper('c1') })
    expect(await screen.findByText('agent_blocked')).toBeInTheDocument()
  })

  it('multiple notifications all rendered', async () => {
    server.use(
      http.get('/api/companies/:id/agents', () => HttpResponse.json([])),
      http.get('/api/companies/:id/notifications', () =>
        HttpResponse.json([
          makeNotification('n1', 'agent_blocked'),
          makeNotification('n2', 'issue_failed'),
          makeNotification('n3', 'escalation'),
        ]),
      ),
    )
    render(<Dashboard />, { wrapper: makeWrapper('c1') })
    expect(await screen.findByText('agent_blocked')).toBeInTheDocument()
    expect(await screen.findByText('issue_failed')).toBeInTheDocument()
    expect(await screen.findByText('escalation')).toBeInTheDocument()
  })

  it('no notifications shows empty state', async () => {
    server.use(
      http.get('/api/companies/:id/agents', () => HttpResponse.json([])),
      http.get('/api/companies/:id/notifications', () => HttpResponse.json([])),
    )
    render(<Dashboard />, { wrapper: makeWrapper('c1') })
    expect(await screen.findByText('No active notifications.')).toBeInTheDocument()
  })

  it('"Dismiss" button present on notification', async () => {
    server.use(
      http.get('/api/companies/:id/agents', () => HttpResponse.json([])),
      http.get('/api/companies/:id/notifications', () =>
        HttpResponse.json([makeNotification('n1', 'escalation')]),
      ),
    )
    render(<Dashboard />, { wrapper: makeWrapper('c1') })
    expect(await screen.findByRole('button', { name: /dismiss/i })).toBeInTheDocument()
  })

  it('"Dismiss" button calls POST /dismiss', async () => {
    let dismissHit = false
    server.use(
      http.get('/api/companies/:id/agents', () => HttpResponse.json([])),
      http.get('/api/companies/:id/notifications', () =>
        HttpResponse.json([makeNotification('n1', 'escalation')]),
      ),
      http.post('/api/companies/:id/notifications/:id/dismiss', () => {
        dismissHit = true
        return HttpResponse.json({ ok: true })
      }),
    )
    render(<Dashboard />, { wrapper: makeWrapper('c1') })
    await userEvent.click(await screen.findByRole('button', { name: /dismiss/i }))
    await waitFor(() => expect(dismissHit).toBe(true))
  })

  it('on dismiss success: re-fetches notifications', async () => {
    let notifCallCount = 0
    server.use(
      http.get('/api/companies/:id/agents', () => HttpResponse.json([])),
      http.get('/api/companies/:id/notifications', () => {
        notifCallCount++
        return HttpResponse.json([makeNotification('n1', 'escalation')])
      }),
      http.post('/api/companies/:id/notifications/:id/dismiss', () =>
        HttpResponse.json({ ok: true }),
      ),
    )
    render(<Dashboard />, { wrapper: makeWrapper('c1') })
    await userEvent.click(await screen.findByRole('button', { name: /dismiss/i }))
    await waitFor(() => expect(notifCallCount).toBeGreaterThan(1))
  })

  it('loading state handled gracefully (no crash)', async () => {
    server.use(
      http.get('/api/companies/:id/agents', () => HttpResponse.json([])),
      http.get('/api/companies/:id/notifications', () => HttpResponse.json([])),
    )
    expect(() => render(<Dashboard />, { wrapper: makeWrapper('c1') })).not.toThrow()
  })

  it('API error handled gracefully (no crash)', async () => {
    server.use(
      http.get('/api/companies/:id/agents', () => HttpResponse.json({}, { status: 500 })),
      http.get('/api/companies/:id/notifications', () => HttpResponse.json({}, { status: 500 })),
    )
    expect(() => render(<Dashboard />, { wrapper: makeWrapper('c1') })).not.toThrow()
  })
})

// ── AgentStatusCard component ─────────────────────────────────────────────────

describe('AgentStatusCard', () => {
  it('renders label', () => {
    render(<AgentStatusCard label="Working" count={3} colour="text-emerald-400" />)
    expect(screen.getByText('Working')).toBeInTheDocument()
  })

  it('renders count', () => {
    render(<AgentStatusCard label="Idle" count={7} colour="text-zinc-300" />)
    expect(screen.getByText('7')).toBeInTheDocument()
  })

  it('renders zero count', () => {
    render(<AgentStatusCard label="Failed" count={0} colour="text-red-400" />)
    expect(screen.getByText('0')).toBeInTheDocument()
  })

  it('renders label and count together', () => {
    render(<AgentStatusCard label="Blocked" count={2} colour="text-amber-400" />)
    expect(screen.getByText('Blocked')).toBeInTheDocument()
    expect(screen.getByText('2')).toBeInTheDocument()
  })
})

// ── EscalationList component ──────────────────────────────────────────────────

describe('EscalationList', () => {
  it('renders notification type', () => {
    const onDismiss = vi.fn()
    render(
      <EscalationList
        notifications={[makeNotification('n1', 'agent_escalated')]}
        onDismiss={onDismiss}
      />
    )
    expect(screen.getByText('agent_escalated')).toBeInTheDocument()
  })

  it('dismiss button present', () => {
    const onDismiss = vi.fn()
    render(
      <EscalationList
        notifications={[makeNotification('n1', 'test')]}
        onDismiss={onDismiss}
      />
    )
    expect(screen.getByRole('button', { name: /dismiss/i })).toBeInTheDocument()
  })

  it('dismiss button calls onDismiss with correct id', async () => {
    const onDismiss = vi.fn()
    render(
      <EscalationList
        notifications={[makeNotification('n-special', 'test')]}
        onDismiss={onDismiss}
      />
    )
    await userEvent.click(screen.getByRole('button', { name: /dismiss/i }))
    expect(onDismiss).toHaveBeenCalledWith('n-special')
  })

  it('empty list shows "No active notifications."', () => {
    render(<EscalationList notifications={[]} onDismiss={vi.fn()} />)
    expect(screen.getByText('No active notifications.')).toBeInTheDocument()
  })

  it('multiple notifications all rendered', () => {
    const onDismiss = vi.fn()
    render(
      <EscalationList
        notifications={[
          makeNotification('n1', 'type_a'),
          makeNotification('n2', 'type_b'),
        ]}
        onDismiss={onDismiss}
      />
    )
    expect(screen.getByText('type_a')).toBeInTheDocument()
    expect(screen.getByText('type_b')).toBeInTheDocument()
    expect(screen.getAllByRole('button', { name: /dismiss/i })).toHaveLength(2)
  })
})

import { vi } from 'vitest'
