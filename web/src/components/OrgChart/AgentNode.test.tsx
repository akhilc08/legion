import { describe, it, expect } from 'vitest'
import { render, screen } from '@testing-library/react'
import { ReactFlowProvider } from '@xyflow/react'
import { AgentNode } from './AgentNode'
import type { Agent } from '@/lib/types'

const mockAgent: Agent = {
  id: 'a1', company_id: 'c1', role: 'cto', title: 'CTO Agent',
  system_prompt: '', manager_id: null, runtime: 'claude_code',
  status: 'working', monthly_budget: 100000, token_spend: 5000,
  chat_token_spend: 200, pid: null, created_at: '', updated_at: '',
}

describe('AgentNode', () => {
  it('renders agent title and status', () => {
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
