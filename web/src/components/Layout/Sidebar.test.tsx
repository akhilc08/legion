import { describe, it, expect, vi, beforeEach } from 'vitest'
import { render, screen, within } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { MemoryRouter } from 'react-router-dom'
import type { Agent } from '@/lib/types'

// Must be hoisted before the module import
vi.mock('./CompanySelector', () => ({
  CompanySelector: () => <div data-testid="company-selector" />,
}))

// Store mock – returns different values based on selector key accessed
const mockSetAgentId = vi.fn()
let mockAgentId: string | null = null

vi.mock('@/store/useAppStore', () => ({
  useAppStore: (sel: (s: {
    companyId: string | null
    agentId: string | null
    setAgentId: (id: string | null) => void
  }) => unknown) =>
    sel({
      companyId: 'company-1',
      agentId: mockAgentId,
      setAgentId: mockSetAgentId,
    }),
}))

import { Sidebar } from './Sidebar'

function makeAgent(overrides: Partial<Agent> = {}): Agent {
  return {
    id: 'a1',
    company_id: 'c1',
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
    created_at: '',
    updated_at: '',
    ...overrides,
  }
}

function renderSidebar(hireBadgeCount = 0, agents: Agent[] = []) {
  return render(
    <MemoryRouter initialEntries={['/companies/c1/dashboard']}>
      <Sidebar hireBadgeCount={hireBadgeCount} agents={agents} />
    </MemoryRouter>
  )
}

beforeEach(() => {
  mockAgentId = null
  mockSetAgentId.mockClear()
})

describe('Sidebar – nav links', () => {
  it('renders Dashboard nav link', () => {
    renderSidebar()
    expect(screen.getByText('Dashboard')).toBeInTheDocument()
  })

  it('renders Org Chart nav link', () => {
    renderSidebar()
    expect(screen.getByText('Org Chart')).toBeInTheDocument()
  })

  it('renders Issues nav link', () => {
    renderSidebar()
    expect(screen.getByText('Issues')).toBeInTheDocument()
  })

  it('renders Hiring nav link', () => {
    renderSidebar()
    expect(screen.getByText('Hiring')).toBeInTheDocument()
  })

  it('renders Audit nav link', () => {
    renderSidebar()
    expect(screen.getByText('Audit')).toBeInTheDocument()
  })

  it('renders Files nav link', () => {
    renderSidebar()
    expect(screen.getByText('Files')).toBeInTheDocument()
  })

  it('renders all 6 nav links together', () => {
    renderSidebar()
    ;['Dashboard', 'Org Chart', 'Issues', 'Hiring', 'Audit', 'Files'].forEach((label) =>
      expect(screen.getByText(label)).toBeInTheDocument()
    )
  })
})

describe('Sidebar – hire badge in nav', () => {
  it('hireBadgeCount=0: no badge shown on Hiring nav link', () => {
    renderSidebar(0)
    // The only "0" text should not appear as a badge – there should be no red badge
    const badges = document.querySelectorAll('.bg-red-600')
    expect(badges).toHaveLength(0)
  })

  it('hireBadgeCount=3: shows badge "3" on Hiring link', () => {
    renderSidebar(3)
    // Find badge within the nav
    const nav = screen.getByRole('navigation')
    expect(within(nav).getByText('3')).toBeInTheDocument()
  })

  it('hireBadgeCount=1: shows badge "1" on Hiring link', () => {
    renderSidebar(1)
    const nav = screen.getByRole('navigation')
    expect(within(nav).getByText('1')).toBeInTheDocument()
  })

  it('hireBadgeCount=10: shows badge "10"', () => {
    renderSidebar(10)
    const nav = screen.getByRole('navigation')
    expect(within(nav).getByText('10')).toBeInTheDocument()
  })
})

describe('Sidebar – agent tree visibility', () => {
  it('agents=[]: no agent tree section rendered', () => {
    renderSidebar(0, [])
    expect(screen.queryByText('Agents')).not.toBeInTheDocument()
  })

  it('agents=[boardAgent]: renders agent tree section', () => {
    const board = makeAgent({ id: 'b1', role: 'board', title: 'Board' })
    renderSidebar(0, [board])
    expect(screen.getByText(/agents/i)).toBeInTheDocument()
  })
})

describe('Sidebar – board agent row', () => {
  it('board agent renders "Board (you)" text instead of title', () => {
    const board = makeAgent({ id: 'b1', role: 'board', title: 'The Board' })
    renderSidebar(0, [board])
    expect(screen.getByText('Board (you)')).toBeInTheDocument()
  })

  it('board agent title is NOT shown verbatim', () => {
    const board = makeAgent({ id: 'b1', role: 'board', title: 'The Board' })
    renderSidebar(0, [board])
    expect(screen.queryByText('The Board')).not.toBeInTheDocument()
  })

  it('board agent shows "human" label', () => {
    const board = makeAgent({ id: 'b1', role: 'board', title: 'Board' })
    renderSidebar(0, [board])
    expect(screen.getByText('human')).toBeInTheDocument()
  })

  it('non-board agent does NOT show "human" label', () => {
    const agent = makeAgent({ id: 'a1', role: 'cto', title: 'CTO' })
    renderSidebar(0, [agent])
    expect(screen.queryByText('human')).not.toBeInTheDocument()
  })
})

describe('Sidebar – normal agent row', () => {
  it('renders agent title', () => {
    const agent = makeAgent({ title: 'Chief Engineer' })
    renderSidebar(0, [agent])
    expect(screen.getByText('Chief Engineer')).toBeInTheDocument()
  })

  it('clicking agent row calls setAgentId with agent id', async () => {
    const agent = makeAgent({ id: 'agent-xyz', title: 'Marketing Lead' })
    renderSidebar(0, [agent])
    await userEvent.click(screen.getByText('Marketing Lead'))
    expect(mockSetAgentId).toHaveBeenCalledWith('agent-xyz')
  })

  it('clicking board agent row calls setAgentId with board id', async () => {
    const board = makeAgent({ id: 'board-id', role: 'board', title: 'Board' })
    renderSidebar(0, [board])
    await userEvent.click(screen.getByText('Board (you)'))
    expect(mockSetAgentId).toHaveBeenCalledWith('board-id')
  })
})

describe('Sidebar – selected agent highlight', () => {
  it('selected agent row has bg-zinc-800 class', () => {
    mockAgentId = 'a1'
    const agent = makeAgent({ id: 'a1', title: 'Selected Agent' })
    renderSidebar(0, [agent])
    const row = screen.getByText('Selected Agent').closest('[class*="rounded-md"]')
    expect(row?.className).toContain('bg-zinc-800')
  })

  it('non-selected agent row does NOT have bg-zinc-800 class', () => {
    mockAgentId = 'other-id'
    const agent = makeAgent({ id: 'a1', title: 'Unselected Agent' })
    renderSidebar(0, [agent])
    const row = screen.getByText('Unselected Agent').closest('[class*="rounded-md"]')
    // Should not have the solid bg-zinc-800 active class (hover variant is bg-zinc-800/50, not bg-zinc-800 directly)
    // Split on spaces to check for the exact token
    const classes = (row?.className ?? '').split(' ')
    expect(classes).not.toContain('bg-zinc-800')
  })
})

describe('Sidebar – chevron and subtree', () => {
  it('agent with children shows chevron button', () => {
    const parent = makeAgent({ id: 'p1', title: 'Parent Agent', manager_id: null })
    const child = makeAgent({ id: 'c1', title: 'Child Agent', manager_id: 'p1' })
    renderSidebar(0, [parent, child])
    // ChevronRight renders inside a button
    const chevronBtn = document.querySelector('button svg')
    expect(chevronBtn).toBeTruthy()
  })

  it('agent without children does NOT show chevron button', () => {
    const agent = makeAgent({ id: 'a1', title: 'Solo Agent', manager_id: null })
    renderSidebar(0, [agent])
    // No children → no chevron button (only the w-3 span placeholder)
    const chevronSvgs = document.querySelectorAll('button svg')
    expect(chevronSvgs).toHaveLength(0)
  })

  it('child agents are rendered when subtree is open (depth < 2)', () => {
    const parent = makeAgent({ id: 'p1', title: 'Parent Agent', manager_id: null })
    const child = makeAgent({ id: 'c1', title: 'Child Agent', manager_id: 'p1' })
    renderSidebar(0, [parent, child])
    // depth=0 < 2, so children start open
    expect(screen.getByText('Child Agent')).toBeInTheDocument()
  })

  it('clicking chevron toggles subtree closed', async () => {
    const parent = makeAgent({ id: 'p1', title: 'Parent Agent', manager_id: null })
    const child = makeAgent({ id: 'c1', title: 'Child Agent', manager_id: 'p1' })
    renderSidebar(0, [parent, child])
    // Initially open
    expect(screen.getByText('Child Agent')).toBeInTheDocument()
    // Click chevron button
    const chevronBtn = document.querySelector('button') as HTMLButtonElement
    await userEvent.click(chevronBtn)
    expect(screen.queryByText('Child Agent')).not.toBeInTheDocument()
  })

  it('clicking chevron twice toggles subtree back open', async () => {
    const parent = makeAgent({ id: 'p1', title: 'Parent Agent', manager_id: null })
    const child = makeAgent({ id: 'c1', title: 'Child Agent', manager_id: 'p1' })
    renderSidebar(0, [parent, child])
    const chevronBtn = document.querySelector('button') as HTMLButtonElement
    await userEvent.click(chevronBtn)
    await userEvent.click(chevronBtn)
    expect(screen.getByText('Child Agent')).toBeInTheDocument()
  })
})

describe('Sidebar – depth-based padding', () => {
  it('depth=0 items have paddingLeft=8px', () => {
    const agent = makeAgent({ id: 'a1', title: 'Root Agent', manager_id: null })
    renderSidebar(0, [agent])
    const row = screen.getByText('Root Agent').closest('[style]') as HTMLElement | null
    expect(row?.style.paddingLeft).toBe('8px')
  })

  it('depth=1 items have paddingLeft=22px (8 + 14)', () => {
    const parent = makeAgent({ id: 'p1', title: 'Root', manager_id: null })
    const child = makeAgent({ id: 'c1', title: 'Child Depth 1', manager_id: 'p1' })
    renderSidebar(0, [parent, child])
    const childRow = screen.getByText('Child Depth 1').closest('[style]') as HTMLElement | null
    expect(childRow?.style.paddingLeft).toBe('22px')
  })
})

describe('Sidebar – agent status dot colours', () => {
  const cases: Array<[Agent['status'], string]> = [
    ['idle', 'bg-zinc-500'],
    ['working', 'bg-emerald-400'],
    ['paused', 'bg-blue-400'],
    ['blocked', 'bg-amber-400'],
    ['failed', 'bg-red-500'],
    ['done', 'bg-zinc-600'],
    ['degraded', 'bg-orange-400'],
  ]

  cases.forEach(([status, expectedClass]) => {
    it(`status=${status} → dot has class ${expectedClass}`, () => {
      const agent = makeAgent({ id: 'a1', title: `Agent-${status}`, status })
      const { container } = renderSidebar(0, [agent])
      // Status dot is a span with rounded-full and the status class
      const dot = container.querySelector(`.${expectedClass}.rounded-full`)
      expect(dot).toBeTruthy()
    })
  })
})

describe('Sidebar – hiring badge in agent row', () => {
  it('hiring agent with hireBadgeCount > 0 shows badge in agent row', () => {
    const hiringAgent = makeAgent({ id: 'h1', role: 'hiring', title: 'Hiring Manager' })
    renderSidebar(5, [hiringAgent])
    // There should be a badge in the agent row area too
    // Two "5" badges: one in nav, one in agent row
    const badges = screen.getAllByText('5')
    expect(badges.length).toBeGreaterThanOrEqual(2)
  })

  it('hiring agent with hireBadgeCount=0 does NOT show badge in agent row', () => {
    const hiringAgent = makeAgent({ id: 'h1', role: 'hiring', title: 'Hiring Manager' })
    renderSidebar(0, [hiringAgent])
    // No red badge should appear
    const redBadges = document.querySelectorAll('.bg-red-600')
    expect(redBadges).toHaveLength(0)
  })
})

describe('Sidebar – roots computation', () => {
  it('agents with no manager_id are roots', () => {
    const a = makeAgent({ id: 'a1', title: 'Agent A', manager_id: null })
    const b = makeAgent({ id: 'b1', title: 'Agent B', manager_id: null })
    renderSidebar(0, [a, b])
    expect(screen.getByText('Agent A')).toBeInTheDocument()
    expect(screen.getByText('Agent B')).toBeInTheDocument()
  })

  it('agent whose manager is not in list is treated as root', () => {
    const orphan = makeAgent({ id: 'o1', title: 'Orphan Agent', manager_id: 'missing-id' })
    renderSidebar(0, [orphan])
    expect(screen.getByText('Orphan Agent')).toBeInTheDocument()
  })

  it('board root is always sorted first', () => {
    const cto = makeAgent({ id: 'c1', role: 'cto', title: 'CTO', manager_id: null })
    const board = makeAgent({ id: 'b1', role: 'board', title: 'Board', manager_id: null })
    renderSidebar(0, [cto, board])
    // Board (you) should appear before CTO in the DOM
    const allItems = screen.getAllByText(/.+/)
    const boardIdx = allItems.findIndex((el) => el.textContent === 'Board (you)')
    const ctoIdx = allItems.findIndex((el) => el.textContent === 'CTO')
    expect(boardIdx).toBeLessThan(ctoIdx)
  })
})

describe('Sidebar – CompanySelector', () => {
  it('renders CompanySelector at the bottom', () => {
    renderSidebar()
    expect(screen.getByTestId('company-selector')).toBeInTheDocument()
  })
})

describe('Sidebar – branding', () => {
  it('renders "legion" branding text', () => {
    renderSidebar()
    expect(screen.getByText('legion')).toBeInTheDocument()
  })
})
