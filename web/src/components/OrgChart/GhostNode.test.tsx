import { describe, it, expect, vi, beforeEach } from 'vitest'
import { render, screen, fireEvent } from '@testing-library/react'
import type { PendingHire } from '@/lib/types'

vi.mock('@xyflow/react', () => ({
  Handle: ({ type }: { type: string; position: string }) => (
    <div data-testid={`handle-${type}`} />
  ),
  Position: { Top: 'top', Bottom: 'bottom' },
}))

import { GhostNode } from './GhostNode'
import type { GhostNodeType } from './GhostNode'
import type { NodeProps } from '@xyflow/react'

const mockHire: PendingHire = {
  id: 'hire-1',
  role_title: 'Senior Engineer',
  status: 'pending',
  company_id: 'co-1',
  requested_by_agent_id: 'agent-1',
  reporting_to_agent_id: 'agent-2',
  system_prompt: 'Do good work',
  runtime: 'claude_code',
  budget_allocation: 1000,
  initial_task: null,
  created_at: '2024-01-01T00:00:00Z',
}

function makeProps(overrides?: Partial<GhostNodeType['data']>): NodeProps<GhostNodeType> {
  const onApprove = vi.fn()
  const onReject = vi.fn()
  return {
    id: 'node-1',
    data: { hire: mockHire, onApprove, onReject, ...overrides },
    type: 'ghostNode',
    selected: false,
    isConnectable: true,
    zIndex: 0,
    xPos: 0,
    yPos: 0,
    dragging: false,
    positionAbsoluteX: 0,
    positionAbsoluteY: 0,
  } as NodeProps<GhostNodeType>
}

describe('GhostNode – content', () => {
  it('renders "Awaiting Approval" text', () => {
    render(<GhostNode {...makeProps()} />)
    expect(screen.getByText('Awaiting Approval')).toBeInTheDocument()
  })

  it('renders the hire role_title', () => {
    render(<GhostNode {...makeProps()} />)
    expect(screen.getByText('Senior Engineer')).toBeInTheDocument()
  })

  it('renders an Approve button', () => {
    render(<GhostNode {...makeProps()} />)
    expect(screen.getByRole('button', { name: /approve/i })).toBeInTheDocument()
  })

  it('renders a Reject button', () => {
    render(<GhostNode {...makeProps()} />)
    expect(screen.getByRole('button', { name: /reject/i })).toBeInTheDocument()
  })

  it('renders target handle', () => {
    render(<GhostNode {...makeProps()} />)
    expect(screen.getByTestId('handle-target')).toBeInTheDocument()
  })

  it('renders source handle', () => {
    render(<GhostNode {...makeProps()} />)
    expect(screen.getByTestId('handle-source')).toBeInTheDocument()
  })
})

describe('GhostNode – styling', () => {
  it('has border-dashed class on the card container', () => {
    const { container } = render(<GhostNode {...makeProps()} />)
    const card = container.querySelector('.border-dashed')
    expect(card).toBeTruthy()
  })

  it('has opacity-70 class on the card container', () => {
    const { container } = render(<GhostNode {...makeProps()} />)
    const card = container.querySelector('.opacity-70')
    expect(card).toBeTruthy()
  })

  it('has bg-zinc-900\\/60 class on the card container', () => {
    const { container } = render(<GhostNode {...makeProps()} />)
    const card = container.querySelector('.bg-zinc-900\\/60')
    expect(card).toBeTruthy()
  })
})

describe('GhostNode – button interactions', () => {
  let onApprove: ReturnType<typeof vi.fn>
  let onReject: ReturnType<typeof vi.fn>

  beforeEach(() => {
    onApprove = vi.fn()
    onReject = vi.fn()
  })

  it('clicking Approve calls onApprove', () => {
    render(<GhostNode {...makeProps({ hire: mockHire, onApprove, onReject })} />)
    fireEvent.click(screen.getByRole('button', { name: /approve/i }))
    expect(onApprove).toHaveBeenCalledOnce()
  })

  it('clicking Reject calls onReject', () => {
    render(<GhostNode {...makeProps({ hire: mockHire, onApprove, onReject })} />)
    fireEvent.click(screen.getByRole('button', { name: /reject/i }))
    expect(onReject).toHaveBeenCalledOnce()
  })

  it('clicking Approve does not call onReject', () => {
    render(<GhostNode {...makeProps({ hire: mockHire, onApprove, onReject })} />)
    fireEvent.click(screen.getByRole('button', { name: /approve/i }))
    expect(onReject).not.toHaveBeenCalled()
  })

  it('clicking Reject does not call onApprove', () => {
    render(<GhostNode {...makeProps({ hire: mockHire, onApprove, onReject })} />)
    fireEvent.click(screen.getByRole('button', { name: /reject/i }))
    expect(onApprove).not.toHaveBeenCalled()
  })

  it('Approve click calls stopPropagation (event does not bubble to container)', () => {
    const containerHandler = vi.fn()
    render(
      <div onClick={containerHandler}>
        <GhostNode {...makeProps({ hire: mockHire, onApprove, onReject })} />
      </div>
    )
    fireEvent.click(screen.getByRole('button', { name: /approve/i }))
    expect(containerHandler).not.toHaveBeenCalled()
  })

  it('Reject click calls stopPropagation (event does not bubble to container)', () => {
    const containerHandler = vi.fn()
    render(
      <div onClick={containerHandler}>
        <GhostNode {...makeProps({ hire: mockHire, onApprove, onReject })} />
      </div>
    )
    fireEvent.click(screen.getByRole('button', { name: /reject/i }))
    expect(containerHandler).not.toHaveBeenCalled()
  })
})

describe('GhostNode – different role titles', () => {
  it('renders a different role title correctly', () => {
    const hire: PendingHire = { ...mockHire, role_title: 'Product Manager' }
    render(<GhostNode {...makeProps({ hire, onApprove: vi.fn(), onReject: vi.fn() })} />)
    expect(screen.getByText('Product Manager')).toBeInTheDocument()
  })

  it('renders yet another role title', () => {
    const hire: PendingHire = { ...mockHire, role_title: 'Design Lead' }
    render(<GhostNode {...makeProps({ hire, onApprove: vi.fn(), onReject: vi.fn() })} />)
    expect(screen.getByText('Design Lead')).toBeInTheDocument()
  })
})
