import { describe, it, expect } from 'vitest'
import { render, screen } from '@testing-library/react'
import { ReactFlowProvider } from '@xyflow/react'
import { AgentNode } from './AgentNode'
import type { Agent } from '@/lib/types'

function makeAgent(overrides: Partial<Agent> = {}): Agent {
  return {
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
    ...overrides,
  }
}

function makeProps(agent: Agent, selected = false): Parameters<typeof AgentNode>[0] {
  return { id: agent.id, data: { agent }, selected } as Parameters<typeof AgentNode>[0]
}

function renderNode(agent: Agent, selected = false) {
  return render(
    <ReactFlowProvider>
      <AgentNode {...makeProps(agent, selected)} />
    </ReactFlowProvider>
  )
}

describe('AgentNode – content', () => {
  it('renders agent title', () => {
    renderNode(makeAgent({ title: 'My Test Agent' }))
    expect(screen.getByText('My Test Agent')).toBeInTheDocument()
  })

  it('renders agent status text', () => {
    renderNode(makeAgent({ status: 'working' }))
    expect(screen.getByText('working')).toBeInTheDocument()
  })

  it('renders correct title for a different agent', () => {
    renderNode(makeAgent({ title: 'Backend Engineer' }))
    expect(screen.getByText('Backend Engineer')).toBeInTheDocument()
  })

  it('renders status text for all statuses', () => {
    const statuses: Agent['status'][] = ['idle', 'working', 'paused', 'blocked', 'failed', 'done', 'degraded']
    statuses.forEach((status) => {
      const { unmount } = renderNode(makeAgent({ status }))
      expect(screen.getByText(status)).toBeInTheDocument()
      unmount()
    })
  })
})

describe('AgentNode – status dot colours', () => {
  it('idle → bg-zinc-400', () => {
    const { container } = renderNode(makeAgent({ status: 'idle' }))
    const dot = container.querySelector('[data-status-dot]')
    expect(dot?.className).toContain('bg-zinc-400')
  })

  it('working → bg-emerald-400', () => {
    const { container } = renderNode(makeAgent({ status: 'working' }))
    const dot = container.querySelector('[data-status-dot]')
    expect(dot?.className).toContain('bg-emerald-400')
  })

  it('paused → bg-blue-400', () => {
    const { container } = renderNode(makeAgent({ status: 'paused' }))
    const dot = container.querySelector('[data-status-dot]')
    expect(dot?.className).toContain('bg-blue-400')
  })

  it('blocked → bg-amber-400', () => {
    const { container } = renderNode(makeAgent({ status: 'blocked' }))
    const dot = container.querySelector('[data-status-dot]')
    expect(dot?.className).toContain('bg-amber-400')
  })

  it('failed → bg-red-400', () => {
    const { container } = renderNode(makeAgent({ status: 'failed' }))
    const dot = container.querySelector('[data-status-dot]')
    expect(dot?.className).toContain('bg-red-400')
  })

  it('done → bg-zinc-600', () => {
    const { container } = renderNode(makeAgent({ status: 'done' }))
    const dot = container.querySelector('[data-status-dot]')
    expect(dot?.className).toContain('bg-zinc-600')
  })

  it('degraded → bg-orange-400', () => {
    const { container } = renderNode(makeAgent({ status: 'degraded' }))
    const dot = container.querySelector('[data-status-dot]')
    expect(dot?.className).toContain('bg-orange-400')
  })
})

describe('AgentNode – data-status-dot attribute', () => {
  it('status dot element has data-status-dot attribute (idle)', () => {
    const { container } = renderNode(makeAgent({ status: 'idle' }))
    expect(container.querySelector('[data-status-dot]')).toBeTruthy()
  })

  it('status dot element has data-status-dot attribute (working)', () => {
    const { container } = renderNode(makeAgent({ status: 'working' }))
    expect(container.querySelector('[data-status-dot]')).toBeTruthy()
  })

  it('status dot element has data-status-dot attribute (failed)', () => {
    const { container } = renderNode(makeAgent({ status: 'failed' }))
    expect(container.querySelector('[data-status-dot]')).toBeTruthy()
  })

  it('exactly one status dot rendered', () => {
    const { container } = renderNode(makeAgent())
    const dots = container.querySelectorAll('[data-status-dot]')
    expect(dots).toHaveLength(1)
  })
})

describe('AgentNode – selected state', () => {
  it('selected=true applies border-blue-500 to container', () => {
    const { container } = renderNode(makeAgent(), true)
    const card = container.querySelector('.border-blue-500')
    expect(card).toBeTruthy()
  })

  it('selected=false does NOT apply border-blue-500', () => {
    const { container } = renderNode(makeAgent(), false)
    const card = container.querySelector('.border-blue-500')
    expect(card).toBeNull()
  })

  it('selected=false applies border-zinc-700 instead', () => {
    const { container } = renderNode(makeAgent(), false)
    const card = container.querySelector('.border-zinc-700')
    expect(card).toBeTruthy()
  })

  it('selected=true does NOT apply border-zinc-700', () => {
    const { container } = renderNode(makeAgent(), true)
    const card = container.querySelector('.border-zinc-700')
    expect(card).toBeNull()
  })
})

describe('AgentNode – handles', () => {
  it('has a target handle (top)', () => {
    const { container } = renderNode(makeAgent())
    // ReactFlow renders handles with data-handlepos
    const topHandle = container.querySelector('[data-handlepos="top"]')
    expect(topHandle).toBeTruthy()
  })

  it('has a source handle (bottom)', () => {
    const { container } = renderNode(makeAgent())
    const bottomHandle = container.querySelector('[data-handlepos="bottom"]')
    expect(bottomHandle).toBeTruthy()
  })
})

describe('AgentNode – title truncation', () => {
  it('title span has truncate class', () => {
    const { container } = renderNode(makeAgent({ title: 'Very Long Title That Should Be Truncated' }))
    const titleEl = screen.getByText('Very Long Title That Should Be Truncated')
    expect(titleEl.className).toContain('truncate')
  })

  it('title span has max-w-[100px] class', () => {
    const { container } = renderNode(makeAgent({ title: 'Another Long Agent Title' }))
    const titleEl = screen.getByText('Another Long Agent Title')
    expect(titleEl.className).toContain('max-w-[100px]')
  })
})

describe('AgentNode – various agents', () => {
  it('renders CEO agent with idle status correctly', () => {
    const { container } = renderNode(makeAgent({ title: 'CEO', status: 'idle' }))
    expect(screen.getByText('CEO')).toBeInTheDocument()
    expect(screen.getByText('idle')).toBeInTheDocument()
    expect(container.querySelector('[data-status-dot]')?.className).toContain('bg-zinc-400')
  })

  it('renders designer agent with paused status correctly', () => {
    const { container } = renderNode(makeAgent({ title: 'Designer', status: 'paused' }))
    expect(screen.getByText('Designer')).toBeInTheDocument()
    expect(container.querySelector('[data-status-dot]')?.className).toContain('bg-blue-400')
  })

  it('renders engineer agent with failed status and selected=true', () => {
    const { container } = renderNode(makeAgent({ title: 'Engineer', status: 'failed' }), true)
    expect(screen.getByText('Engineer')).toBeInTheDocument()
    expect(container.querySelector('[data-status-dot]')?.className).toContain('bg-red-400')
    expect(container.querySelector('.border-blue-500')).toBeTruthy()
  })

  it('renders agent with degraded status and selected=false', () => {
    const { container } = renderNode(makeAgent({ title: 'Monitor', status: 'degraded' }), false)
    expect(container.querySelector('[data-status-dot]')?.className).toContain('bg-orange-400')
    expect(container.querySelector('.border-blue-500')).toBeNull()
  })

  it('renders agent with done status', () => {
    const { container } = renderNode(makeAgent({ title: 'Retired', status: 'done' }))
    expect(screen.getByText('done')).toBeInTheDocument()
    expect(container.querySelector('[data-status-dot]')?.className).toContain('bg-zinc-600')
  })

  it('renders agent with blocked status', () => {
    const { container } = renderNode(makeAgent({ title: 'Stuck Agent', status: 'blocked' }))
    expect(screen.getByText('blocked')).toBeInTheDocument()
    expect(container.querySelector('[data-status-dot]')?.className).toContain('bg-amber-400')
  })
})
