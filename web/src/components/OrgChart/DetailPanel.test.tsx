import { describe, it, expect, beforeAll, afterEach, afterAll, vi } from 'vitest'
import { render, screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { http, HttpResponse } from 'msw'
import { server } from '@/test/server'
import type { Agent, Issue } from '@/lib/types'
import { DetailPanel } from './DetailPanel'

const COMPANY_ID = 'company-1'

vi.mock('@/store/useAppStore', () => ({
  useAppStore: (sel: (s: { companyId: string }) => unknown) =>
    sel({ companyId: COMPANY_ID }),
}))

beforeAll(() => server.listen())
afterEach(() => server.resetHandlers())
afterAll(() => server.close())

function makeAgent(overrides: Partial<Agent> = {}): Agent {
  return {
    id: 'agent-1',
    company_id: COMPANY_ID,
    role: 'cto',
    title: 'CTO Agent',
    system_prompt: '',
    manager_id: null,
    runtime: 'claude_code',
    status: 'idle',
    monthly_budget: 100000,
    token_spend: 5000,
    chat_token_spend: 200,
    pid: null,
    created_at: '',
    updated_at: '',
    ...overrides,
  }
}

function makeIssue(overrides: Partial<Issue> = {}): Issue {
  return {
    id: 'issue-1',
    company_id: COMPANY_ID,
    title: 'Fix the bug',
    description: 'Something is broken',
    assignee_id: 'agent-1',
    parent_id: null,
    status: 'in_progress',
    output_path: null,
    attempt_count: 2,
    last_failure_reason: null,
    escalation_id: null,
    created_at: '',
    updated_at: '',
    ...overrides,
  }
}

function makeQC() {
  return new QueryClient({
    defaultOptions: { queries: { retry: false }, mutations: { retry: false } },
  })
}

function renderPanel(agent: Agent, onClose = vi.fn()) {
  return render(
    <QueryClientProvider client={makeQC()}>
      <DetailPanel agent={agent} onClose={onClose} />
    </QueryClientProvider>
  )
}

function emptyIssues() {
  return http.get(`/api/companies/${COMPANY_ID}/issues`, () => HttpResponse.json([]))
}

describe('DetailPanel – content rendering', () => {
  it('renders agent title in heading', () => {
    server.use(emptyIssues())
    renderPanel(makeAgent({ title: 'Engineering Lead' }))
    expect(screen.getByRole('heading', { name: 'Engineering Lead' })).toBeInTheDocument()
  })

  it('renders agent title as h3', () => {
    server.use(emptyIssues())
    renderPanel(makeAgent({ title: 'My Agent' }))
    expect(screen.getByText('My Agent').tagName).toBe('H3')
  })

  it('renders agent status text', () => {
    server.use(emptyIssues())
    renderPanel(makeAgent({ status: 'working' }))
    expect(screen.getByText(/working/)).toBeInTheDocument()
  })

  it('renders agent runtime', () => {
    server.use(emptyIssues())
    renderPanel(makeAgent({ runtime: 'claude_code' }))
    expect(screen.getByText('claude_code')).toBeInTheDocument()
  })

  it('renders agent runtime for openclaw', () => {
    server.use(emptyIssues())
    renderPanel(makeAgent({ runtime: 'openclaw' }))
    expect(screen.getByText('openclaw')).toBeInTheDocument()
  })
})

describe('DetailPanel – close button', () => {
  it('calls onClose when X button is clicked', async () => {
    server.use(emptyIssues())
    const onClose = vi.fn()
    renderPanel(makeAgent(), onClose)
    const closeBtn = screen.getByRole('button', { name: '' }) // X button has no aria-label
    // Find X button by its SVG or by finding the button that's not one of the action buttons
    const buttons = screen.getAllByRole('button')
    // The X button is the first / standalone close button
    const xBtn = buttons.find((b) => b.querySelector('svg') && !b.textContent?.trim())
    expect(xBtn).toBeTruthy()
    await userEvent.click(xBtn!)
    expect(onClose).toHaveBeenCalledOnce()
  })

  it('close button is present in the rendered panel', () => {
    server.use(emptyIssues())
    renderPanel(makeAgent())
    // X icon renders as an svg within a button
    const closeButtons = document.querySelectorAll('button svg')
    expect(closeButtons.length).toBeGreaterThan(0)
  })
})

describe('DetailPanel – Spawn button', () => {
  it('shows Spawn button when status=idle', () => {
    server.use(emptyIssues())
    renderPanel(makeAgent({ status: 'idle' }))
    expect(screen.getByRole('button', { name: /spawn/i })).toBeInTheDocument()
  })

  it('does NOT show Spawn button when status=working', () => {
    server.use(emptyIssues())
    renderPanel(makeAgent({ status: 'working' }))
    expect(screen.queryByRole('button', { name: /spawn/i })).not.toBeInTheDocument()
  })

  it('does NOT show Spawn button when status=paused', () => {
    server.use(emptyIssues())
    renderPanel(makeAgent({ status: 'paused' }))
    expect(screen.queryByRole('button', { name: /spawn/i })).not.toBeInTheDocument()
  })

  it('does NOT show Spawn button when status=failed', () => {
    server.use(emptyIssues())
    renderPanel(makeAgent({ status: 'failed' }))
    expect(screen.queryByRole('button', { name: /spawn/i })).not.toBeInTheDocument()
  })

  it('does NOT show Spawn button when status=done', () => {
    server.use(emptyIssues())
    renderPanel(makeAgent({ status: 'done' }))
    expect(screen.queryByRole('button', { name: /spawn/i })).not.toBeInTheDocument()
  })

  it('clicking Spawn calls POST /spawn endpoint', async () => {
    let spawnCalled = false
    server.use(
      emptyIssues(),
      http.post(`/api/companies/${COMPANY_ID}/agents/agent-1/spawn`, () => {
        spawnCalled = true
        return HttpResponse.json({})
      })
    )
    renderPanel(makeAgent({ status: 'idle' }))
    await userEvent.click(screen.getByRole('button', { name: /spawn/i }))
    await waitFor(() => expect(spawnCalled).toBe(true))
  })
})

describe('DetailPanel – Kill button', () => {
  it('shows Kill button when status=working', () => {
    server.use(emptyIssues())
    renderPanel(makeAgent({ status: 'working' }))
    expect(screen.getByRole('button', { name: /kill/i })).toBeInTheDocument()
  })

  it('shows Kill button when status=paused', () => {
    server.use(emptyIssues())
    renderPanel(makeAgent({ status: 'paused' }))
    expect(screen.getByRole('button', { name: /kill/i })).toBeInTheDocument()
  })

  it('shows Kill button when status=failed', () => {
    server.use(emptyIssues())
    renderPanel(makeAgent({ status: 'failed' }))
    expect(screen.getByRole('button', { name: /kill/i })).toBeInTheDocument()
  })

  it('shows Kill button when status=blocked', () => {
    server.use(emptyIssues())
    renderPanel(makeAgent({ status: 'blocked' }))
    expect(screen.getByRole('button', { name: /kill/i })).toBeInTheDocument()
  })

  it('shows Kill button when status=degraded', () => {
    server.use(emptyIssues())
    renderPanel(makeAgent({ status: 'degraded' }))
    expect(screen.getByRole('button', { name: /kill/i })).toBeInTheDocument()
  })

  it('does NOT show Kill button when status=idle', () => {
    server.use(emptyIssues())
    renderPanel(makeAgent({ status: 'idle' }))
    expect(screen.queryByRole('button', { name: /kill/i })).not.toBeInTheDocument()
  })

  it('does NOT show Kill button when status=done', () => {
    server.use(emptyIssues())
    renderPanel(makeAgent({ status: 'done' }))
    expect(screen.queryByRole('button', { name: /kill/i })).not.toBeInTheDocument()
  })

  it('clicking Kill calls POST /kill endpoint', async () => {
    let killCalled = false
    server.use(
      emptyIssues(),
      http.post(`/api/companies/${COMPANY_ID}/agents/agent-1/kill`, () => {
        killCalled = true
        return HttpResponse.json({})
      })
    )
    renderPanel(makeAgent({ status: 'working' }))
    await userEvent.click(screen.getByRole('button', { name: /kill/i }))
    await waitFor(() => expect(killCalled).toBe(true))
  })
})

describe('DetailPanel – Pause/Resume button', () => {
  it('shows "Pause" button when status=working', () => {
    server.use(emptyIssues())
    renderPanel(makeAgent({ status: 'working' }))
    expect(screen.getByRole('button', { name: /pause/i })).toBeInTheDocument()
    expect(screen.getByRole('button', { name: /pause/i })).toHaveTextContent('Pause')
  })

  it('shows "Resume" button when status=paused', () => {
    server.use(emptyIssues())
    renderPanel(makeAgent({ status: 'paused' }))
    expect(screen.getByRole('button', { name: /resume/i })).toBeInTheDocument()
    expect(screen.getByRole('button', { name: /resume/i })).toHaveTextContent('Resume')
  })

  it('does NOT show Pause/Resume button when status=idle', () => {
    server.use(emptyIssues())
    renderPanel(makeAgent({ status: 'idle' }))
    expect(screen.queryByRole('button', { name: /pause/i })).not.toBeInTheDocument()
    expect(screen.queryByRole('button', { name: /resume/i })).not.toBeInTheDocument()
  })

  it('does NOT show Pause/Resume button when status=done', () => {
    server.use(emptyIssues())
    renderPanel(makeAgent({ status: 'done' }))
    expect(screen.queryByRole('button', { name: /pause/i })).not.toBeInTheDocument()
    expect(screen.queryByRole('button', { name: /resume/i })).not.toBeInTheDocument()
  })

  it('clicking Pause (when working) calls POST /pause', async () => {
    let pauseCalled = false
    server.use(
      emptyIssues(),
      http.post(`/api/companies/${COMPANY_ID}/agents/agent-1/pause`, () => {
        pauseCalled = true
        return HttpResponse.json({})
      })
    )
    renderPanel(makeAgent({ status: 'working' }))
    await userEvent.click(screen.getByRole('button', { name: /pause/i }))
    await waitFor(() => expect(pauseCalled).toBe(true))
  })

  it('clicking Resume (when paused) calls POST /resume', async () => {
    let resumeCalled = false
    server.use(
      emptyIssues(),
      http.post(`/api/companies/${COMPANY_ID}/agents/agent-1/resume`, () => {
        resumeCalled = true
        return HttpResponse.json({})
      })
    )
    renderPanel(makeAgent({ status: 'paused' }))
    await userEvent.click(screen.getByRole('button', { name: /resume/i }))
    await waitFor(() => expect(resumeCalled).toBe(true))
  })
})

describe('DetailPanel – token budget', () => {
  it('shows "∞" when monthly_budget=0', () => {
    server.use(emptyIssues())
    renderPanel(makeAgent({ monthly_budget: 0, token_spend: 0 }))
    expect(screen.getByText('∞')).toBeInTheDocument()
  })

  it('shows "∞" when monthly_budget=2147483647', () => {
    server.use(emptyIssues())
    renderPanel(makeAgent({ monthly_budget: 2147483647, token_spend: 0 }))
    expect(screen.getByText('∞')).toBeInTheDocument()
  })

  it('shows percentage when budget is finite', () => {
    server.use(emptyIssues())
    // 50000 / 100000 = 50%
    renderPanel(makeAgent({ monthly_budget: 100000, token_spend: 50000 }))
    expect(screen.getByText('50%')).toBeInTheDocument()
  })

  it('shows 0% when no tokens spent', () => {
    server.use(emptyIssues())
    renderPanel(makeAgent({ monthly_budget: 100000, token_spend: 0 }))
    expect(screen.getByText('0%')).toBeInTheDocument()
  })

  it('shows 100% when fully spent', () => {
    server.use(emptyIssues())
    renderPanel(makeAgent({ monthly_budget: 100000, token_spend: 100000 }))
    expect(screen.getByText('100%')).toBeInTheDocument()
  })

  it('budget bar > 80% shows bg-red-500', () => {
    server.use(emptyIssues())
    const { container } = renderPanel(makeAgent({ monthly_budget: 100, token_spend: 85 }))
    // 85% > 80 → red
    expect(container.querySelector('.bg-red-500')).toBeTruthy()
  })

  it('budget bar 51-80% shows bg-amber-500', () => {
    server.use(emptyIssues())
    const { container } = renderPanel(makeAgent({ monthly_budget: 100, token_spend: 70 }))
    // 70% is in 51-80 range → amber
    expect(container.querySelector('.bg-amber-500')).toBeTruthy()
  })

  it('budget bar <= 50% shows bg-emerald-500', () => {
    server.use(emptyIssues())
    const { container } = renderPanel(makeAgent({ monthly_budget: 100, token_spend: 50 }))
    // 50% → emerald
    expect(container.querySelector('.bg-emerald-500')).toBeTruthy()
  })

  it('budget bar 0% shows bg-emerald-500', () => {
    server.use(emptyIssues())
    const { container } = renderPanel(makeAgent({ monthly_budget: 100, token_spend: 0 }))
    expect(container.querySelector('.bg-emerald-500')).toBeTruthy()
  })

  it('budget bar width style matches percentage', () => {
    server.use(emptyIssues())
    const { container } = renderPanel(makeAgent({ monthly_budget: 100, token_spend: 25 }))
    const bar = container.querySelector('.bg-emerald-500') as HTMLElement | null
    expect(bar?.style.width).toBe('25%')
  })

  it('token spend formatted with toLocaleString', () => {
    server.use(emptyIssues())
    renderPanel(makeAgent({ monthly_budget: 1000000, token_spend: 12345 }))
    // toLocaleString of 12345 → "12,345" in en-US
    expect(screen.getByText(/12[,.]345/)).toBeInTheDocument()
  })

  it('unlimited budget shows "∞ / ∞ tokens" or similar', () => {
    server.use(emptyIssues())
    renderPanel(makeAgent({ monthly_budget: 0, token_spend: 0 }))
    // Multiple ∞ elements exist (percentage span + token line), use getAllByText
    const infinityEls = screen.getAllByText(/∞/)
    expect(infinityEls.length).toBeGreaterThanOrEqual(1)
  })
})

describe('DetailPanel – current issue', () => {
  it('shows current in_progress issue assigned to this agent', async () => {
    const issue = makeIssue({ assignee_id: 'agent-1', status: 'in_progress', title: 'Active Bug Fix' })
    server.use(
      http.get(`/api/companies/${COMPANY_ID}/issues`, () => HttpResponse.json([issue]))
    )
    renderPanel(makeAgent({ id: 'agent-1' }))
    await waitFor(() =>
      expect(screen.getByText('Active Bug Fix')).toBeInTheDocument()
    )
  })

  it('shows "Current issue" label when issue exists', async () => {
    const issue = makeIssue({ assignee_id: 'agent-1', status: 'in_progress' })
    server.use(
      http.get(`/api/companies/${COMPANY_ID}/issues`, () => HttpResponse.json([issue]))
    )
    renderPanel(makeAgent({ id: 'agent-1' }))
    await waitFor(() =>
      expect(screen.getByText('Current issue')).toBeInTheDocument()
    )
  })

  it('shows attempt count for current issue', async () => {
    const issue = makeIssue({ assignee_id: 'agent-1', status: 'in_progress', attempt_count: 3 })
    server.use(
      http.get(`/api/companies/${COMPANY_ID}/issues`, () => HttpResponse.json([issue]))
    )
    renderPanel(makeAgent({ id: 'agent-1' }))
    await waitFor(() =>
      expect(screen.getByText('attempt #3')).toBeInTheDocument()
    )
  })

  it('does NOT show issue section when no current issue', async () => {
    server.use(
      http.get(`/api/companies/${COMPANY_ID}/issues`, () => HttpResponse.json([]))
    )
    renderPanel(makeAgent({ id: 'agent-1' }))
    await waitFor(() => {
      expect(screen.queryByText('Current issue')).not.toBeInTheDocument()
    })
  })

  it('does NOT show issue assigned to a different agent', async () => {
    const issue = makeIssue({ assignee_id: 'other-agent', status: 'in_progress', title: 'Someone Else Bug' })
    server.use(
      http.get(`/api/companies/${COMPANY_ID}/issues`, () => HttpResponse.json([issue]))
    )
    renderPanel(makeAgent({ id: 'agent-1' }))
    await waitFor(() => {
      expect(screen.queryByText('Someone Else Bug')).not.toBeInTheDocument()
    })
  })

  it('does NOT show issue with non-in_progress status', async () => {
    const issue = makeIssue({ assignee_id: 'agent-1', status: 'done', title: 'Completed Task' })
    server.use(
      http.get(`/api/companies/${COMPANY_ID}/issues`, () => HttpResponse.json([issue]))
    )
    renderPanel(makeAgent({ id: 'agent-1' }))
    await waitFor(() => {
      expect(screen.queryByText('Completed Task')).not.toBeInTheDocument()
    })
  })
})

describe('DetailPanel – status colours', () => {
  const statusColours: Array<[Agent['status'], string]> = [
    ['idle', 'text-zinc-400'],
    ['working', 'text-emerald-400'],
    ['paused', 'text-blue-400'],
    ['blocked', 'text-amber-400'],
    ['failed', 'text-red-400'],
    ['done', 'text-zinc-500'],
    ['degraded', 'text-orange-400'],
  ]

  statusColours.forEach(([status, colorClass]) => {
    it(`status=${status} renders with ${colorClass}`, () => {
      server.use(emptyIssues())
      const { container } = renderPanel(makeAgent({ status }))
      const statusEl = container.querySelector(`.${colorClass}`)
      expect(statusEl).toBeTruthy()
    })
  })
})
