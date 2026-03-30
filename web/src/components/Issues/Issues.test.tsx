import { describe, it, expect, beforeAll, afterEach, afterAll, vi } from 'vitest'
import { render, screen, waitFor, within } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { MemoryRouter, Route, Routes } from 'react-router-dom'
import { server } from '@/test/server'
import { http, HttpResponse } from 'msw'
import { Issues } from './index'
import { IssueRow } from './IssueRow'
import type { Issue, Agent } from '@/lib/types'

beforeAll(() => server.listen())
afterEach(() => server.resetHandlers())
afterAll(() => server.close())

function makeWrapper(companyId: string) {
  const qc = new QueryClient({ defaultOptions: { queries: { retry: false }, mutations: { retry: false } } })
  return ({ children }: { children: React.ReactNode }) => (
    <QueryClientProvider client={qc}>
      <MemoryRouter initialEntries={[`/companies/${companyId}/issues`]}>
        <Routes><Route path="/companies/:companyId/issues" element={<>{children}</>} /></Routes>
      </MemoryRouter>
    </QueryClientProvider>
  )
}

const mockIssues: Issue[] = [
  {
    id: 'i1', company_id: 'c1', title: 'Fix login bug', description: 'Login is broken',
    assignee_id: 'a1', parent_id: null, status: 'in_progress', output_path: null,
    attempt_count: 2, last_failure_reason: null, escalation_id: null,
    created_at: new Date().toISOString(), updated_at: new Date().toISOString(),
  },
  {
    id: 'i2', company_id: 'c1', title: 'Deploy to staging', description: '',
    assignee_id: null, parent_id: null, status: 'pending', output_path: null,
    attempt_count: 0, last_failure_reason: null, escalation_id: null,
    created_at: new Date().toISOString(), updated_at: new Date().toISOString(),
  },
  {
    id: 'i3', company_id: 'c1', title: 'Blocked task', description: '',
    assignee_id: null, parent_id: null, status: 'blocked', output_path: null,
    attempt_count: 1, last_failure_reason: 'Network timeout', escalation_id: null,
    created_at: new Date().toISOString(), updated_at: new Date().toISOString(),
  },
]

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

function setupDefaults() {
  server.use(
    http.get('/api/companies/:id/agents', () => HttpResponse.json(mockAgents)),
    http.get('/api/companies/:id/issues', () => HttpResponse.json(mockIssues)),
  )
}

// ── Heading ──────────────────────────────────────────────────────────────────

describe('Issues heading', () => {
  it('renders "Issues" heading', async () => {
    setupDefaults()
    render(<Issues />, { wrapper: makeWrapper('c1') })
    expect(await screen.findByText('Issues')).toBeInTheDocument()
  })
})

// ── Issue list rendering ──────────────────────────────────────────────────────

describe('Issues list rendering', () => {
  it('shows issue titles from API', async () => {
    setupDefaults()
    render(<Issues />, { wrapper: makeWrapper('c1') })
    expect(await screen.findByText('Fix login bug')).toBeInTheDocument()
    expect(await screen.findByText('Deploy to staging')).toBeInTheDocument()
  })

  it('shows empty state "No issues found." when API returns []', async () => {
    server.use(
      http.get('/api/companies/:id/agents', () => HttpResponse.json([])),
      http.get('/api/companies/:id/issues', () => HttpResponse.json([])),
    )
    render(<Issues />, { wrapper: makeWrapper('c1') })
    expect(await screen.findByText('No issues found.')).toBeInTheDocument()
  })

  it('issues fetched on mount', async () => {
    let fetched = false
    server.use(
      http.get('/api/companies/:id/agents', () => HttpResponse.json([])),
      http.get('/api/companies/:id/issues', () => { fetched = true; return HttpResponse.json([]) }),
    )
    render(<Issues />, { wrapper: makeWrapper('c1') })
    await waitFor(() => expect(fetched).toBe(true))
  })

  it('status badges rendered for each issue', async () => {
    setupDefaults()
    render(<Issues />, { wrapper: makeWrapper('c1') })
    // Use findAllByText to handle the filter <option> duplicates
    expect((await screen.findAllByText('in_progress')).length).toBeGreaterThanOrEqual(1)
    expect((await screen.findAllByText('pending')).length).toBeGreaterThanOrEqual(1)
    expect((await screen.findAllByText('blocked')).length).toBeGreaterThanOrEqual(1)
  })
})

// ── Search filtering ──────────────────────────────────────────────────────────

describe('Issues search filtering', () => {
  it('search input filters issues by title (case insensitive match)', async () => {
    setupDefaults()
    render(<Issues />, { wrapper: makeWrapper('c1') })
    await screen.findByText('Fix login bug')
    const searchInput = screen.getByPlaceholderText('Search issues…')
    await userEvent.type(searchInput, 'login')
    expect(screen.getByText('Fix login bug')).toBeInTheDocument()
    expect(screen.queryByText('Deploy to staging')).not.toBeInTheDocument()
  })

  it('search is case insensitive', async () => {
    setupDefaults()
    render(<Issues />, { wrapper: makeWrapper('c1') })
    await screen.findByText('Fix login bug')
    const searchInput = screen.getByPlaceholderText('Search issues…')
    await userEvent.type(searchInput, 'LOGIN')
    expect(screen.getByText('Fix login bug')).toBeInTheDocument()
  })

  it('search shows empty state when no match', async () => {
    setupDefaults()
    render(<Issues />, { wrapper: makeWrapper('c1') })
    await screen.findByText('Fix login bug')
    const searchInput = screen.getByPlaceholderText('Search issues…')
    await userEvent.type(searchInput, 'zzznomatch')
    expect(await screen.findByText('No issues found.')).toBeInTheDocument()
  })
})

// ── Status filter ─────────────────────────────────────────────────────────────

describe('Issues status filter', () => {
  it('status filter dropdown filters issues', async () => {
    setupDefaults()
    render(<Issues />, { wrapper: makeWrapper('c1') })
    await screen.findByText('Fix login bug')
    const select = screen.getByRole('combobox')
    await userEvent.selectOptions(select, 'pending')
    expect(screen.getByText('Deploy to staging')).toBeInTheDocument()
    expect(screen.queryByText('Fix login bug')).not.toBeInTheDocument()
  })

  it('"All statuses" option shows all issues', async () => {
    setupDefaults()
    render(<Issues />, { wrapper: makeWrapper('c1') })
    await screen.findByText('Fix login bug')
    const select = screen.getByRole('combobox')
    await userEvent.selectOptions(select, 'pending')
    await userEvent.selectOptions(select, '')
    expect(screen.getByText('Fix login bug')).toBeInTheDocument()
    expect(screen.getByText('Deploy to staging')).toBeInTheDocument()
  })

  it('each status option is available: pending', async () => {
    setupDefaults()
    render(<Issues />, { wrapper: makeWrapper('c1') })
    await screen.findByText('Fix login bug')
    expect(screen.getByRole('option', { name: 'pending' })).toBeInTheDocument()
  })

  it('each status option is available: in_progress', async () => {
    setupDefaults()
    render(<Issues />, { wrapper: makeWrapper('c1') })
    await screen.findByText('Fix login bug')
    expect(screen.getByRole('option', { name: 'in_progress' })).toBeInTheDocument()
  })

  it('each status option is available: blocked', async () => {
    setupDefaults()
    render(<Issues />, { wrapper: makeWrapper('c1') })
    await screen.findByText('Fix login bug')
    expect(screen.getByRole('option', { name: 'blocked' })).toBeInTheDocument()
  })

  it('each status option is available: done', async () => {
    setupDefaults()
    render(<Issues />, { wrapper: makeWrapper('c1') })
    await screen.findByText('Fix login bug')
    expect(screen.getByRole('option', { name: 'done' })).toBeInTheDocument()
  })

  it('each status option is available: failed', async () => {
    setupDefaults()
    render(<Issues />, { wrapper: makeWrapper('c1') })
    await screen.findByText('Fix login bug')
    expect(screen.getByRole('option', { name: 'failed' })).toBeInTheDocument()
  })

  it('combined search + status filter', async () => {
    const extraIssues: Issue[] = [
      ...mockIssues,
      {
        id: 'i4', company_id: 'c1', title: 'Fix auth bug', description: '',
        assignee_id: null, parent_id: null, status: 'pending', output_path: null,
        attempt_count: 0, last_failure_reason: null, escalation_id: null,
        created_at: new Date().toISOString(), updated_at: new Date().toISOString(),
      },
    ]
    server.use(
      http.get('/api/companies/:id/agents', () => HttpResponse.json(mockAgents)),
      http.get('/api/companies/:id/issues', () => HttpResponse.json(extraIssues)),
    )
    render(<Issues />, { wrapper: makeWrapper('c1') })
    await screen.findByText('Fix login bug')
    const searchInput = screen.getByPlaceholderText('Search issues…')
    await userEvent.type(searchInput, 'Fix')
    const select = screen.getByRole('combobox')
    await userEvent.selectOptions(select, 'pending')
    expect(screen.getByText('Fix auth bug')).toBeInTheDocument()
    expect(screen.queryByText('Fix login bug')).not.toBeInTheDocument()
    expect(screen.queryByText('Deploy to staging')).not.toBeInTheDocument()
  })
})

// ── New Issue form ────────────────────────────────────────────────────────────

describe('New Issue form visibility', () => {
  it('"New Issue" button shows form when clicked', async () => {
    setupDefaults()
    render(<Issues />, { wrapper: makeWrapper('c1') })
    await screen.findByText('Issues')
    await userEvent.click(screen.getByRole('button', { name: 'New Issue' }))
    expect(screen.getByPlaceholderText('Title')).toBeInTheDocument()
  })

  it('"Cancel" button hides form', async () => {
    setupDefaults()
    render(<Issues />, { wrapper: makeWrapper('c1') })
    await screen.findByText('Issues')
    await userEvent.click(screen.getByRole('button', { name: 'New Issue' }))
    expect(screen.getByPlaceholderText('Title')).toBeInTheDocument()
    await userEvent.click(screen.getByRole('button', { name: 'Cancel' }))
    expect(screen.queryByPlaceholderText('Title')).not.toBeInTheDocument()
  })

  it('form has Title input', async () => {
    setupDefaults()
    render(<Issues />, { wrapper: makeWrapper('c1') })
    await screen.findByText('Issues')
    await userEvent.click(screen.getByRole('button', { name: 'New Issue' }))
    expect(screen.getByPlaceholderText('Title')).toBeInTheDocument()
  })

  it('form has Description textarea', async () => {
    setupDefaults()
    render(<Issues />, { wrapper: makeWrapper('c1') })
    await screen.findByText('Issues')
    await userEvent.click(screen.getByRole('button', { name: 'New Issue' }))
    expect(screen.getByPlaceholderText('Description (optional)')).toBeInTheDocument()
  })

  it('form has Assignee select', async () => {
    setupDefaults()
    render(<Issues />, { wrapper: makeWrapper('c1') })
    await screen.findByText('Issues')
    await userEvent.click(screen.getByRole('button', { name: 'New Issue' }))
    expect(screen.getByRole('option', { name: 'No assignee' })).toBeInTheDocument()
  })

  it('form has Parent issue select', async () => {
    setupDefaults()
    render(<Issues />, { wrapper: makeWrapper('c1') })
    await screen.findByText('Fix login bug')
    await userEvent.click(screen.getByRole('button', { name: 'New Issue' }))
    expect(screen.getByRole('option', { name: 'No parent issue' })).toBeInTheDocument()
  })
})

// ── Submit button state ───────────────────────────────────────────────────────

describe('New Issue form submit state', () => {
  it('Submit button disabled when title is empty', async () => {
    setupDefaults()
    render(<Issues />, { wrapper: makeWrapper('c1') })
    await screen.findByText('Issues')
    await userEvent.click(screen.getByRole('button', { name: 'New Issue' }))
    expect(screen.getByRole('button', { name: 'Create Issue' })).toBeDisabled()
  })

  it('Submit button enabled when title filled', async () => {
    setupDefaults()
    render(<Issues />, { wrapper: makeWrapper('c1') })
    await screen.findByText('Issues')
    await userEvent.click(screen.getByRole('button', { name: 'New Issue' }))
    await userEvent.type(screen.getByPlaceholderText('Title'), 'My new issue')
    expect(screen.getByRole('button', { name: 'Create Issue' })).not.toBeDisabled()
  })
})

// ── Form submission ───────────────────────────────────────────────────────────

describe('New Issue form submission', () => {
  it('submitting form calls POST /issues with correct payload', async () => {
    let capturedBody: unknown = null
    server.use(
      http.get('/api/companies/:id/agents', () => HttpResponse.json(mockAgents)),
      http.get('/api/companies/:id/issues', () => HttpResponse.json(mockIssues)),
      http.post('/api/companies/:id/issues', async ({ request }) => {
        capturedBody = await request.json()
        return HttpResponse.json({ id: 'new-1', ...capturedBody as object })
      }),
    )
    render(<Issues />, { wrapper: makeWrapper('c1') })
    await screen.findByText('Issues')
    await userEvent.click(screen.getByRole('button', { name: 'New Issue' }))
    await userEvent.type(screen.getByPlaceholderText('Title'), 'Test Issue')
    await userEvent.type(screen.getByPlaceholderText('Description (optional)'), 'A description')
    await userEvent.click(screen.getByRole('button', { name: 'Create Issue' }))
    await waitFor(() => expect(capturedBody).toMatchObject({ title: 'Test Issue', description: 'A description' }))
  })

  it('on success: form resets and closes', async () => {
    server.use(
      http.get('/api/companies/:id/agents', () => HttpResponse.json(mockAgents)),
      http.get('/api/companies/:id/issues', () => HttpResponse.json(mockIssues)),
      http.post('/api/companies/:id/issues', () => HttpResponse.json({ id: 'new-1' })),
    )
    render(<Issues />, { wrapper: makeWrapper('c1') })
    await screen.findByText('Issues')
    await userEvent.click(screen.getByRole('button', { name: 'New Issue' }))
    await userEvent.type(screen.getByPlaceholderText('Title'), 'Test Issue')
    await userEvent.click(screen.getByRole('button', { name: 'Create Issue' }))
    await waitFor(() => expect(screen.queryByPlaceholderText('Title')).not.toBeInTheDocument())
  })

  it('on error: shows error message', async () => {
    server.use(
      http.get('/api/companies/:id/agents', () => HttpResponse.json(mockAgents)),
      http.get('/api/companies/:id/issues', () => HttpResponse.json(mockIssues)),
      http.post('/api/companies/:id/issues', () => HttpResponse.json({ error: 'fail' }, { status: 500 })),
    )
    render(<Issues />, { wrapper: makeWrapper('c1') })
    await screen.findByText('Issues')
    await userEvent.click(screen.getByRole('button', { name: 'New Issue' }))
    await userEvent.type(screen.getByPlaceholderText('Title'), 'Fail Issue')
    await userEvent.click(screen.getByRole('button', { name: 'Create Issue' }))
    await waitFor(() => expect(screen.getByText(/request failed/i)).toBeInTheDocument())
  })

  it('"Creating…" shown while mutation pending', async () => {
    let resolve: () => void
    const pending = new Promise<void>((res) => { resolve = res })
    server.use(
      http.get('/api/companies/:id/agents', () => HttpResponse.json(mockAgents)),
      http.get('/api/companies/:id/issues', () => HttpResponse.json(mockIssues)),
      http.post('/api/companies/:id/issues', () => pending.then(() => HttpResponse.json({}))),
    )
    render(<Issues />, { wrapper: makeWrapper('c1') })
    await screen.findByText('Issues')
    await userEvent.click(screen.getByRole('button', { name: 'New Issue' }))
    await userEvent.type(screen.getByPlaceholderText('Title'), 'Slow Issue')
    await userEvent.click(screen.getByRole('button', { name: 'Create Issue' }))
    expect(await screen.findByText('Creating…')).toBeInTheDocument()
    resolve!()
  })

  it('API error in create shows error text', async () => {
    server.use(
      http.get('/api/companies/:id/agents', () => HttpResponse.json([])),
      http.get('/api/companies/:id/issues', () => HttpResponse.json([])),
      http.post('/api/companies/:id/issues', () => HttpResponse.json({ error: 'server error' }, { status: 500 })),
    )
    render(<Issues />, { wrapper: makeWrapper('c1') })
    await screen.findByText('Issues')
    await userEvent.click(screen.getByRole('button', { name: 'New Issue' }))
    await userEvent.type(screen.getByPlaceholderText('Title'), 'Error Issue')
    await userEvent.click(screen.getByRole('button', { name: 'Create Issue' }))
    await waitFor(() => expect(screen.getByText(/request failed/i)).toBeInTheDocument())
  })
})

// ── Assignee select ───────────────────────────────────────────────────────────

describe('New Issue form assignee select', () => {
  it('Assignee select shows agents from API', async () => {
    setupDefaults()
    render(<Issues />, { wrapper: makeWrapper('c1') })
    await screen.findByText('Fix login bug')
    await userEvent.click(screen.getByRole('button', { name: 'New Issue' }))
    expect(await screen.findByRole('option', { name: 'Alice' })).toBeInTheDocument()
    expect(await screen.findByRole('option', { name: 'Board Agent' })).toBeInTheDocument()
  })

  it('board agent sorted first in assignee select', async () => {
    setupDefaults()
    render(<Issues />, { wrapper: makeWrapper('c1') })
    await screen.findByText('Fix login bug')
    await userEvent.click(screen.getByRole('button', { name: 'New Issue' }))
    await screen.findByRole('option', { name: 'Board Agent' })
    const selects = screen.getAllByRole('combobox')
    // The assignee select is the first combobox inside the form
    const assigneeSelect = selects.find(s => within(s).queryByText('No assignee') !== null)
    expect(assigneeSelect).toBeDefined()
    const options = within(assigneeSelect!).getAllByRole('option')
    expect(options[1].textContent).toBe('Board Agent')
  })

  it('Parent issue select shows existing issues', async () => {
    setupDefaults()
    render(<Issues />, { wrapper: makeWrapper('c1') })
    await screen.findByText('Fix login bug')
    await userEvent.click(screen.getByRole('button', { name: 'New Issue' }))
    expect(await screen.findByRole('option', { name: 'Fix login bug' })).toBeInTheDocument()
  })
})

// ── IssueRow component ────────────────────────────────────────────────────────

describe('IssueRow', () => {
  function renderRow(issue: Partial<Issue>, agents: Agent[] = []) {
    const fullIssue: Issue = {
      id: 'row1', company_id: 'c1', title: 'Row Issue', description: '',
      assignee_id: null, parent_id: null, status: 'pending', output_path: null,
      attempt_count: 0, last_failure_reason: null, escalation_id: null,
      created_at: new Date().toISOString(), updated_at: new Date().toISOString(),
      ...issue,
    }
    return render(
      <table><tbody><IssueRow issue={fullIssue} agents={agents} /></tbody></table>
    )
  }

  it('renders issue title', () => {
    renderRow({ title: 'My Important Issue' })
    expect(screen.getByText('My Important Issue')).toBeInTheDocument()
  })

  it('renders issue status', () => {
    renderRow({ status: 'blocked' })
    expect(screen.getByText('blocked')).toBeInTheDocument()
  })

  it('renders attempt count', () => {
    renderRow({ attempt_count: 5 })
    expect(screen.getByText('5')).toBeInTheDocument()
  })

  it('shows assignee name when agent found', () => {
    renderRow({ assignee_id: 'a1' }, mockAgents)
    expect(screen.getByText('Alice')).toBeInTheDocument()
  })

  it('shows "—" when no assignee', () => {
    renderRow({ assignee_id: null }, mockAgents)
    expect(screen.getByText('—')).toBeInTheDocument()
  })

  it('shows "—" when assignee_id does not match any agent', () => {
    renderRow({ assignee_id: 'unknown-id' }, mockAgents)
    expect(screen.getByText('—')).toBeInTheDocument()
  })

  it('renders in_progress status with correct text', () => {
    renderRow({ status: 'in_progress' })
    expect(screen.getByText('in_progress')).toBeInTheDocument()
  })

  it('renders done status', () => {
    renderRow({ status: 'done' })
    expect(screen.getByText('done')).toBeInTheDocument()
  })

  it('renders failed status', () => {
    renderRow({ status: 'failed' })
    expect(screen.getByText('failed')).toBeInTheDocument()
  })

  it('renders pending status', () => {
    renderRow({ status: 'pending' })
    expect(screen.getByText('pending')).toBeInTheDocument()
  })

  it('renders attempt_count of 0', () => {
    renderRow({ attempt_count: 0 })
    expect(screen.getByText('0')).toBeInTheDocument()
  })
})
