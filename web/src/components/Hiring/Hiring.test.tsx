import { describe, it, expect, beforeAll, afterEach, afterAll } from 'vitest'
import { render, screen, waitFor, within } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { MemoryRouter, Route, Routes } from 'react-router-dom'
import { server } from '@/test/server'
import { http, HttpResponse } from 'msw'
import { Hiring } from './index'
import { HireCard } from './HireCard'
import type { PendingHire, Agent } from '@/lib/types'

beforeAll(() => server.listen())
afterEach(() => server.resetHandlers())
afterAll(() => server.close())

function makeWrapper(companyId: string) {
  const qc = new QueryClient({ defaultOptions: { queries: { retry: false }, mutations: { retry: false } } })
  return ({ children }: { children: React.ReactNode }) => (
    <QueryClientProvider client={qc}>
      <MemoryRouter initialEntries={[`/companies/${companyId}/hiring`]}>
        <Routes><Route path="/companies/:companyId/hiring" element={<>{children}</>} /></Routes>
      </MemoryRouter>
    </QueryClientProvider>
  )
}

const mockHire: PendingHire = {
  id: 'h1', company_id: 'c1', requested_by_agent_id: 'a1',
  role_title: 'Senior Engineer', reporting_to_agent_id: 'a2',
  system_prompt: 'You are a senior engineer.', runtime: 'claude_code',
  budget_allocation: 50000, initial_task: 'Build auth module',
  status: 'pending', created_at: new Date().toISOString(),
}

const mockHire2: PendingHire = {
  id: 'h2', company_id: 'c1', requested_by_agent_id: 'a1',
  role_title: 'Junior Designer', reporting_to_agent_id: 'a2',
  system_prompt: '', runtime: 'openclaw',
  budget_allocation: 20000, initial_task: null,
  status: 'pending', created_at: new Date().toISOString(),
}

const mockAgents: Agent[] = [
  {
    id: 'a1', company_id: 'c1', role: 'engineer', title: 'Alice',
    system_prompt: '', manager_id: null, runtime: 'claude_code',
    status: 'working', monthly_budget: 10000, token_spend: 0,
    chat_token_spend: 0, pid: null,
    created_at: new Date().toISOString(), updated_at: new Date().toISOString(),
  },
  {
    id: 'a2', company_id: 'c1', role: 'board', title: 'Board Agent',
    system_prompt: '', manager_id: null, runtime: 'claude_code',
    status: 'idle', monthly_budget: 50000, token_spend: 0,
    chat_token_spend: 0, pid: null,
    created_at: new Date().toISOString(), updated_at: new Date().toISOString(),
  },
]

function setupDefaults(hires: PendingHire[] = [mockHire]) {
  server.use(
    http.get('/api/companies/:id/hires', () => HttpResponse.json(hires)),
    http.get('/api/companies/:id/agents', () => HttpResponse.json(mockAgents)),
  )
}

// ── Heading & counts ──────────────────────────────────────────────────────────

describe('Hiring heading', () => {
  it('renders "Hiring" heading', async () => {
    setupDefaults()
    render(<Hiring />, { wrapper: makeWrapper('c1') })
    expect(await screen.findByText('Hiring')).toBeInTheDocument()
  })

  it('shows pending count "(2 pending)" when 2 pending hires', async () => {
    setupDefaults([mockHire, mockHire2])
    render(<Hiring />, { wrapper: makeWrapper('c1') })
    expect(await screen.findByText('(2 pending)')).toBeInTheDocument()
  })

  it('shows "(1 pending)" when 1 pending hire', async () => {
    setupDefaults([mockHire])
    render(<Hiring />, { wrapper: makeWrapper('c1') })
    expect(await screen.findByText('(1 pending)')).toBeInTheDocument()
  })

  it('no pending count shown when 0 pending', async () => {
    server.use(
      http.get('/api/companies/:id/hires', () => HttpResponse.json([])),
      http.get('/api/companies/:id/agents', () => HttpResponse.json(mockAgents)),
    )
    render(<Hiring />, { wrapper: makeWrapper('c1') })
    await screen.findByText('Hiring')
    expect(screen.queryByText(/\d+ pending/i)).not.toBeInTheDocument()
  })

  it('"No pending hire requests." shown when list empty', async () => {
    server.use(
      http.get('/api/companies/:id/hires', () => HttpResponse.json([])),
      http.get('/api/companies/:id/agents', () => HttpResponse.json(mockAgents)),
    )
    render(<Hiring />, { wrapper: makeWrapper('c1') })
    expect(await screen.findByText('No pending hire requests.')).toBeInTheDocument()
  })
})

// ── HireCard rendering ────────────────────────────────────────────────────────

describe('HireCard rendering', () => {
  it('each pending hire rendered as HireCard', async () => {
    setupDefaults([mockHire, mockHire2])
    render(<Hiring />, { wrapper: makeWrapper('c1') })
    expect(await screen.findByText('Senior Engineer')).toBeInTheDocument()
    expect(await screen.findByText('Junior Designer')).toBeInTheDocument()
  })

  it('HireCard shows role_title', async () => {
    setupDefaults()
    render(<Hiring />, { wrapper: makeWrapper('c1') })
    expect(await screen.findByText('Senior Engineer')).toBeInTheDocument()
  })

  it('HireCard shows runtime', async () => {
    setupDefaults()
    render(<Hiring />, { wrapper: makeWrapper('c1') })
    expect(await screen.findByText('claude_code')).toBeInTheDocument()
  })

  it('HireCard shows budget_allocation', async () => {
    setupDefaults()
    render(<Hiring />, { wrapper: makeWrapper('c1') })
    expect(await screen.findByText(/50,000/)).toBeInTheDocument()
  })

  it('HireCard shows initial_task when present', async () => {
    setupDefaults()
    render(<Hiring />, { wrapper: makeWrapper('c1') })
    expect(await screen.findByText('Build auth module')).toBeInTheDocument()
  })

  it('HireCard does not show initial_task section when null', async () => {
    setupDefaults([mockHire2])
    render(<Hiring />, { wrapper: makeWrapper('c1') })
    await screen.findByText('Junior Designer')
    expect(screen.queryByText('Initial task')).not.toBeInTheDocument()
  })
})

// ── HireCard direct unit tests ────────────────────────────────────────────────

describe('HireCard component', () => {
  function renderCard(hire: PendingHire, overrides: Partial<{ isApproving: boolean; isRejecting: boolean }> = {}) {
    const onApprove = vi.fn()
    const onReject = vi.fn()
    render(
      <HireCard
        hire={hire}
        onApprove={onApprove}
        onReject={onReject}
        isApproving={overrides.isApproving ?? false}
        isRejecting={overrides.isRejecting ?? false}
      />
    )
    return { onApprove, onReject }
  }

  it('shows Approve button', () => {
    renderCard(mockHire)
    expect(screen.getByRole('button', { name: /approve/i })).toBeInTheDocument()
  })

  it('shows Reject button', () => {
    renderCard(mockHire)
    expect(screen.getByRole('button', { name: /reject/i })).toBeInTheDocument()
  })

  it('Approve button calls onApprove', async () => {
    const { onApprove } = renderCard(mockHire)
    await userEvent.click(screen.getByRole('button', { name: /approve/i }))
    expect(onApprove).toHaveBeenCalledTimes(1)
  })

  it('Reject button calls onReject', async () => {
    const { onReject } = renderCard(mockHire)
    await userEvent.click(screen.getByRole('button', { name: /reject/i }))
    expect(onReject).toHaveBeenCalledTimes(1)
  })

  it('Approve button shows "Approving…" when isApproving', () => {
    renderCard(mockHire, { isApproving: true })
    expect(screen.getByRole('button', { name: /approving/i })).toBeDisabled()
  })

  it('Reject button shows "Rejecting…" when isRejecting', () => {
    renderCard(mockHire, { isRejecting: true })
    expect(screen.getByRole('button', { name: /rejecting/i })).toBeDisabled()
  })
})

// ── Approve / Reject API calls ────────────────────────────────────────────────

describe('Hiring approve/reject', () => {
  it('HireCard Approve button calls POST /approve', async () => {
    let approveHit = false
    server.use(
      http.get('/api/companies/:id/hires', () => HttpResponse.json([mockHire])),
      http.get('/api/companies/:id/agents', () => HttpResponse.json(mockAgents)),
      http.post('/api/companies/:id/hires/:hireId/approve', () => {
        approveHit = true
        return HttpResponse.json({ status: 'approved' })
      }),
    )
    render(<Hiring />, { wrapper: makeWrapper('c1') })
    await userEvent.click(await screen.findByRole('button', { name: /^Approve$/i }))
    await waitFor(() => expect(approveHit).toBe(true))
  })

  it('HireCard Reject button calls POST /reject', async () => {
    let rejectHit = false
    server.use(
      http.get('/api/companies/:id/hires', () => HttpResponse.json([mockHire])),
      http.get('/api/companies/:id/agents', () => HttpResponse.json(mockAgents)),
      http.post('/api/companies/:id/hires/:hireId/reject', () => {
        rejectHit = true
        return HttpResponse.json({ status: 'rejected' })
      }),
    )
    render(<Hiring />, { wrapper: makeWrapper('c1') })
    await userEvent.click(await screen.findByRole('button', { name: /^Reject$/i }))
    await waitFor(() => expect(rejectHit).toBe(true))
  })

  it('on approve success: refetches hires', async () => {
    let hiresCallCount = 0
    server.use(
      http.get('/api/companies/:id/hires', () => { hiresCallCount++; return HttpResponse.json([mockHire]) }),
      http.get('/api/companies/:id/agents', () => HttpResponse.json(mockAgents)),
      http.post('/api/companies/:id/hires/:hireId/approve', () => HttpResponse.json({ status: 'approved' })),
    )
    render(<Hiring />, { wrapper: makeWrapper('c1') })
    await userEvent.click(await screen.findByRole('button', { name: /^Approve$/i }))
    await waitFor(() => expect(hiresCallCount).toBeGreaterThan(1))
  })

  it('on reject success: refetches hires', async () => {
    let hiresCallCount = 0
    server.use(
      http.get('/api/companies/:id/hires', () => { hiresCallCount++; return HttpResponse.json([mockHire]) }),
      http.get('/api/companies/:id/agents', () => HttpResponse.json(mockAgents)),
      http.post('/api/companies/:id/hires/:hireId/reject', () => HttpResponse.json({ status: 'rejected' })),
    )
    render(<Hiring />, { wrapper: makeWrapper('c1') })
    await userEvent.click(await screen.findByRole('button', { name: /^Reject$/i }))
    await waitFor(() => expect(hiresCallCount).toBeGreaterThan(1))
  })
})

// ── Request Hire form ─────────────────────────────────────────────────────────

describe('Request Hire form visibility', () => {
  it('"Request Hire" button opens form', async () => {
    setupDefaults()
    render(<Hiring />, { wrapper: makeWrapper('c1') })
    await screen.findByText('Hiring')
    await userEvent.click(screen.getByRole('button', { name: 'Request Hire' }))
    expect(screen.getByPlaceholderText('Role title')).toBeInTheDocument()
  })

  it('"Cancel" button closes form', async () => {
    setupDefaults()
    render(<Hiring />, { wrapper: makeWrapper('c1') })
    await screen.findByText('Hiring')
    await userEvent.click(screen.getByRole('button', { name: 'Request Hire' }))
    expect(screen.getByPlaceholderText('Role title')).toBeInTheDocument()
    await userEvent.click(screen.getByRole('button', { name: 'Cancel' }))
    expect(screen.queryByPlaceholderText('Role title')).not.toBeInTheDocument()
  })

  it('form has Role title input', async () => {
    setupDefaults()
    render(<Hiring />, { wrapper: makeWrapper('c1') })
    await screen.findByText('Hiring')
    await userEvent.click(screen.getByRole('button', { name: 'Request Hire' }))
    expect(screen.getByPlaceholderText('Role title')).toBeInTheDocument()
  })

  it('form has Requested by select', async () => {
    setupDefaults()
    render(<Hiring />, { wrapper: makeWrapper('c1') })
    await screen.findByText('Hiring')
    await userEvent.click(screen.getByRole('button', { name: 'Request Hire' }))
    expect(screen.getByRole('option', { name: 'Requested by…' })).toBeInTheDocument()
  })

  it('form has Reports to select', async () => {
    setupDefaults()
    render(<Hiring />, { wrapper: makeWrapper('c1') })
    await screen.findByText('Hiring')
    await userEvent.click(screen.getByRole('button', { name: 'Request Hire' }))
    expect(screen.getByRole('option', { name: 'Reports to…' })).toBeInTheDocument()
  })

  it('form has Runtime select', async () => {
    setupDefaults()
    render(<Hiring />, { wrapper: makeWrapper('c1') })
    await screen.findByText('Hiring')
    await userEvent.click(screen.getByRole('button', { name: 'Request Hire' }))
    expect(screen.getByRole('option', { name: 'claude_code' })).toBeInTheDocument()
  })

  it('form has Budget input', async () => {
    setupDefaults()
    render(<Hiring />, { wrapper: makeWrapper('c1') })
    await screen.findByText('Hiring')
    await userEvent.click(screen.getByRole('button', { name: 'Request Hire' }))
    expect(screen.getByPlaceholderText(/monthly budget/i)).toBeInTheDocument()
  })

  it('form has System prompt textarea', async () => {
    setupDefaults()
    render(<Hiring />, { wrapper: makeWrapper('c1') })
    await screen.findByText('Hiring')
    await userEvent.click(screen.getByRole('button', { name: 'Request Hire' }))
    expect(screen.getByPlaceholderText('System prompt')).toBeInTheDocument()
  })

  it('form has Initial task input', async () => {
    setupDefaults()
    render(<Hiring />, { wrapper: makeWrapper('c1') })
    await screen.findByText('Hiring')
    await userEvent.click(screen.getByRole('button', { name: 'Request Hire' }))
    expect(screen.getByPlaceholderText('Initial task (optional)')).toBeInTheDocument()
  })
})

// ── Submit button state ───────────────────────────────────────────────────────

describe('Request Hire form submit state', () => {
  async function openForm() {
    setupDefaults()
    render(<Hiring />, { wrapper: makeWrapper('c1') })
    await screen.findByText('Hiring')
    await userEvent.click(screen.getByRole('button', { name: 'Request Hire' }))
    await screen.findAllByRole('option', { name: 'Board Agent' })
  }

  it('Submit button disabled when role_title empty', async () => {
    await openForm()
    // select agents but leave role_title empty
    const selects = screen.getAllByRole('combobox')
    const requestedBySelect = selects.find(s => within(s).queryByText('Requested by…') !== null)
    const reportsToSelect = selects.find(s => within(s).queryByText('Reports to…') !== null)
    await userEvent.selectOptions(requestedBySelect!, 'a1')
    await userEvent.selectOptions(reportsToSelect!, 'a2')
    expect(screen.getByRole('button', { name: 'Submit Request' })).toBeDisabled()
  })

  it('Submit button disabled when requested_by_agent_id empty', async () => {
    await openForm()
    await userEvent.type(screen.getByPlaceholderText('Role title'), 'My Role')
    const selects = screen.getAllByRole('combobox')
    const reportsToSelect = selects.find(s => within(s).queryByText('Reports to…') !== null)
    await userEvent.selectOptions(reportsToSelect!, 'a2')
    expect(screen.getByRole('button', { name: 'Submit Request' })).toBeDisabled()
  })

  it('Submit button disabled when reporting_to_agent_id empty', async () => {
    await openForm()
    await userEvent.type(screen.getByPlaceholderText('Role title'), 'My Role')
    const selects = screen.getAllByRole('combobox')
    const requestedBySelect = selects.find(s => within(s).queryByText('Requested by…') !== null)
    await userEvent.selectOptions(requestedBySelect!, 'a1')
    expect(screen.getByRole('button', { name: 'Submit Request' })).toBeDisabled()
  })

  it('Submit button enabled when all required fields filled', async () => {
    await openForm()
    await userEvent.type(screen.getByPlaceholderText('Role title'), 'My Role')
    const selects = screen.getAllByRole('combobox')
    const requestedBySelect = selects.find(s => within(s).queryByText('Requested by…') !== null)
    const reportsToSelect = selects.find(s => within(s).queryByText('Reports to…') !== null)
    await userEvent.selectOptions(requestedBySelect!, 'a1')
    await userEvent.selectOptions(reportsToSelect!, 'a2')
    expect(screen.getByRole('button', { name: 'Submit Request' })).not.toBeDisabled()
  })
})

// ── Form submission ───────────────────────────────────────────────────────────

describe('Request Hire form submission', () => {
  it('submitting calls POST /hires with correct payload', async () => {
    let capturedBody: unknown = null
    server.use(
      http.get('/api/companies/:id/hires', () => HttpResponse.json([])),
      http.get('/api/companies/:id/agents', () => HttpResponse.json(mockAgents)),
      http.post('/api/companies/:id/hires', async ({ request }) => {
        capturedBody = await request.json()
        return HttpResponse.json({ id: 'new-h' })
      }),
    )
    render(<Hiring />, { wrapper: makeWrapper('c1') })
    await screen.findByText('Hiring')
    await userEvent.click(screen.getByRole('button', { name: 'Request Hire' }))
    await screen.findAllByRole('option', { name: 'Board Agent' })
    await userEvent.type(screen.getByPlaceholderText('Role title'), 'New Role')
    const selects = screen.getAllByRole('combobox')
    const requestedBySelect = selects.find(s => within(s).queryByText('Requested by…') !== null)
    const reportsToSelect = selects.find(s => within(s).queryByText('Reports to…') !== null)
    await userEvent.selectOptions(requestedBySelect!, 'a1')
    await userEvent.selectOptions(reportsToSelect!, 'a2')
    await userEvent.click(screen.getByRole('button', { name: 'Submit Request' }))
    await waitFor(() => expect(capturedBody).toMatchObject({ role_title: 'New Role', requested_by_agent_id: 'a1', reporting_to_agent_id: 'a2' }))
  })

  it('"Submitting…" shown while pending', async () => {
    let resolve: () => void
    const pending = new Promise<void>((res) => { resolve = res })
    server.use(
      http.get('/api/companies/:id/hires', () => HttpResponse.json([])),
      http.get('/api/companies/:id/agents', () => HttpResponse.json(mockAgents)),
      http.post('/api/companies/:id/hires', () => pending.then(() => HttpResponse.json({}))),
    )
    render(<Hiring />, { wrapper: makeWrapper('c1') })
    await screen.findByText('Hiring')
    await userEvent.click(screen.getByRole('button', { name: 'Request Hire' }))
    await screen.findAllByRole('option', { name: 'Board Agent' })
    await userEvent.type(screen.getByPlaceholderText('Role title'), 'Slow Role')
    const selects = screen.getAllByRole('combobox')
    const requestedBySelect = selects.find(s => within(s).queryByText('Requested by…') !== null)
    const reportsToSelect = selects.find(s => within(s).queryByText('Reports to…') !== null)
    await userEvent.selectOptions(requestedBySelect!, 'a1')
    await userEvent.selectOptions(reportsToSelect!, 'a2')
    await userEvent.click(screen.getByRole('button', { name: 'Submit Request' }))
    expect(await screen.findByText('Submitting…')).toBeInTheDocument()
    resolve!()
  })

  it('budget empty string sends 0 to API', async () => {
    let capturedBody: unknown = null
    server.use(
      http.get('/api/companies/:id/hires', () => HttpResponse.json([])),
      http.get('/api/companies/:id/agents', () => HttpResponse.json(mockAgents)),
      http.post('/api/companies/:id/hires', async ({ request }) => {
        capturedBody = await request.json()
        return HttpResponse.json({ id: 'new-h' })
      }),
    )
    render(<Hiring />, { wrapper: makeWrapper('c1') })
    await screen.findByText('Hiring')
    await userEvent.click(screen.getByRole('button', { name: 'Request Hire' }))
    await screen.findAllByRole('option', { name: 'Board Agent' })
    await userEvent.type(screen.getByPlaceholderText('Role title'), 'Budget Role')
    const selects = screen.getAllByRole('combobox')
    const requestedBySelect = selects.find(s => within(s).queryByText('Requested by…') !== null)
    const reportsToSelect = selects.find(s => within(s).queryByText('Reports to…') !== null)
    await userEvent.selectOptions(requestedBySelect!, 'a1')
    await userEvent.selectOptions(reportsToSelect!, 'a2')
    await userEvent.click(screen.getByRole('button', { name: 'Submit Request' }))
    await waitFor(() => expect((capturedBody as { budget_allocation: number }).budget_allocation).toBe(0))
  })

  it('budget "50000" sends 50000 to API', async () => {
    let capturedBody: unknown = null
    server.use(
      http.get('/api/companies/:id/hires', () => HttpResponse.json([])),
      http.get('/api/companies/:id/agents', () => HttpResponse.json(mockAgents)),
      http.post('/api/companies/:id/hires', async ({ request }) => {
        capturedBody = await request.json()
        return HttpResponse.json({ id: 'new-h' })
      }),
    )
    render(<Hiring />, { wrapper: makeWrapper('c1') })
    await screen.findByText('Hiring')
    await userEvent.click(screen.getByRole('button', { name: 'Request Hire' }))
    await screen.findAllByRole('option', { name: 'Board Agent' })
    await userEvent.type(screen.getByPlaceholderText('Role title'), 'Paid Role')
    await userEvent.type(screen.getByPlaceholderText(/monthly budget/i), '50000')
    const selects = screen.getAllByRole('combobox')
    const requestedBySelect = selects.find(s => within(s).queryByText('Requested by…') !== null)
    const reportsToSelect = selects.find(s => within(s).queryByText('Reports to…') !== null)
    await userEvent.selectOptions(requestedBySelect!, 'a1')
    await userEvent.selectOptions(reportsToSelect!, 'a2')
    await userEvent.click(screen.getByRole('button', { name: 'Submit Request' }))
    await waitFor(() => expect((capturedBody as { budget_allocation: number }).budget_allocation).toBe(50000))
  })

  it('on success: form resets and closes', async () => {
    server.use(
      http.get('/api/companies/:id/hires', () => HttpResponse.json([])),
      http.get('/api/companies/:id/agents', () => HttpResponse.json(mockAgents)),
      http.post('/api/companies/:id/hires', () => HttpResponse.json({ id: 'new-h' })),
    )
    render(<Hiring />, { wrapper: makeWrapper('c1') })
    await screen.findByText('Hiring')
    await userEvent.click(screen.getByRole('button', { name: 'Request Hire' }))
    await screen.findAllByRole('option', { name: 'Board Agent' })
    await userEvent.type(screen.getByPlaceholderText('Role title'), 'New Role')
    const selects = screen.getAllByRole('combobox')
    const requestedBySelect = selects.find(s => within(s).queryByText('Requested by…') !== null)
    const reportsToSelect = selects.find(s => within(s).queryByText('Reports to…') !== null)
    await userEvent.selectOptions(requestedBySelect!, 'a1')
    await userEvent.selectOptions(reportsToSelect!, 'a2')
    await userEvent.click(screen.getByRole('button', { name: 'Submit Request' }))
    await waitFor(() => expect(screen.queryByPlaceholderText('Role title')).not.toBeInTheDocument())
  })

  it('create.isError shows error message', async () => {
    server.use(
      http.get('/api/companies/:id/hires', () => HttpResponse.json([])),
      http.get('/api/companies/:id/agents', () => HttpResponse.json(mockAgents)),
      http.post('/api/companies/:id/hires', () => HttpResponse.json({ error: 'bad' }, { status: 500 })),
    )
    render(<Hiring />, { wrapper: makeWrapper('c1') })
    await screen.findByText('Hiring')
    await userEvent.click(screen.getByRole('button', { name: 'Request Hire' }))
    await screen.findAllByRole('option', { name: 'Board Agent' })
    await userEvent.type(screen.getByPlaceholderText('Role title'), 'Error Role')
    const selects = screen.getAllByRole('combobox')
    const requestedBySelect = selects.find(s => within(s).queryByText('Requested by…') !== null)
    const reportsToSelect = selects.find(s => within(s).queryByText('Reports to…') !== null)
    await userEvent.selectOptions(requestedBySelect!, 'a1')
    await userEvent.selectOptions(reportsToSelect!, 'a2')
    await userEvent.click(screen.getByRole('button', { name: 'Submit Request' }))
    await waitFor(() => expect(screen.getByText(/request failed/i)).toBeInTheDocument())
  })
})

// ── Runtime select options ────────────────────────────────────────────────────

describe('Runtime select', () => {
  it('runtime select has claude_code option', async () => {
    setupDefaults()
    render(<Hiring />, { wrapper: makeWrapper('c1') })
    await screen.findByText('Hiring')
    await userEvent.click(screen.getByRole('button', { name: 'Request Hire' }))
    expect(screen.getByRole('option', { name: 'claude_code' })).toBeInTheDocument()
  })

  it('runtime select has openclaw option', async () => {
    setupDefaults()
    render(<Hiring />, { wrapper: makeWrapper('c1') })
    await screen.findByText('Hiring')
    await userEvent.click(screen.getByRole('button', { name: 'Request Hire' }))
    expect(screen.getByRole('option', { name: 'openclaw' })).toBeInTheDocument()
  })
})

// ── Agent selects ─────────────────────────────────────────────────────────────

describe('Agent selects in form', () => {
  it('agents list populated in selects', async () => {
    setupDefaults()
    render(<Hiring />, { wrapper: makeWrapper('c1') })
    await screen.findByText('Hiring')
    await userEvent.click(screen.getByRole('button', { name: 'Request Hire' }))
    expect(await screen.findAllByRole('option', { name: 'Alice' })).toHaveLength(2)
  })

  it('board agent sorted first in agent selects', async () => {
    setupDefaults()
    render(<Hiring />, { wrapper: makeWrapper('c1') })
    await screen.findByText('Hiring')
    await userEvent.click(screen.getByRole('button', { name: 'Request Hire' }))
    await screen.findAllByRole('option', { name: 'Board Agent' })
    const selects = screen.getAllByRole('combobox')
    const requestedBySelect = selects.find(s => within(s).queryByText('Requested by…') !== null)
    expect(requestedBySelect).toBeDefined()
    const options = within(requestedBySelect!).getAllByRole('option')
    expect(options[1].textContent).toBe('Board Agent')
  })
})

// need vi import for HireCard tests
import { vi } from 'vitest'
