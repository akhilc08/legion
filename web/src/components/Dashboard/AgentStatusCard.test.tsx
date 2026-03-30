import { describe, it, expect } from 'vitest'
import { render, screen } from '@testing-library/react'
import { AgentStatusCard } from './AgentStatusCard'

describe('AgentStatusCard – label rendering', () => {
  it('renders the label prop', () => {
    render(<AgentStatusCard label="Active" count={5} colour="text-emerald-400" />)
    expect(screen.getByText('Active')).toBeInTheDocument()
  })

  it('renders a different label', () => {
    render(<AgentStatusCard label="Idle" count={0} colour="text-zinc-400" />)
    expect(screen.getByText('Idle')).toBeInTheDocument()
  })

  it('renders label with spaces', () => {
    render(<AgentStatusCard label="In Progress" count={3} colour="text-blue-400" />)
    expect(screen.getByText('In Progress')).toBeInTheDocument()
  })
})

describe('AgentStatusCard – count rendering', () => {
  it('renders the count prop', () => {
    render(<AgentStatusCard label="Active" count={7} colour="text-emerald-400" />)
    expect(screen.getByText('7')).toBeInTheDocument()
  })

  it('renders count=0 as "0"', () => {
    render(<AgentStatusCard label="Blocked" count={0} colour="text-amber-400" />)
    expect(screen.getByText('0')).toBeInTheDocument()
  })

  it('renders count=99', () => {
    render(<AgentStatusCard label="Working" count={99} colour="text-blue-400" />)
    expect(screen.getByText('99')).toBeInTheDocument()
  })

  it('renders count=1', () => {
    render(<AgentStatusCard label="Failed" count={1} colour="text-red-400" />)
    expect(screen.getByText('1')).toBeInTheDocument()
  })

  it('renders large count values', () => {
    render(<AgentStatusCard label="Total" count={1000} colour="text-zinc-100" />)
    expect(screen.getByText('1000')).toBeInTheDocument()
  })
})

describe('AgentStatusCard – colour prop', () => {
  it('applies colour class to the count element', () => {
    render(<AgentStatusCard label="Active" count={5} colour="text-emerald-400" />)
    const countEl = screen.getByText('5')
    expect(countEl.className).toContain('text-emerald-400')
  })

  it('applies red colour class', () => {
    render(<AgentStatusCard label="Failed" count={2} colour="text-red-400" />)
    const countEl = screen.getByText('2')
    expect(countEl.className).toContain('text-red-400')
  })

  it('applies blue colour class', () => {
    render(<AgentStatusCard label="Working" count={4} colour="text-blue-400" />)
    const countEl = screen.getByText('4')
    expect(countEl.className).toContain('text-blue-400')
  })

  it('applies amber colour class', () => {
    render(<AgentStatusCard label="Blocked" count={1} colour="text-amber-400" />)
    const countEl = screen.getByText('1')
    expect(countEl.className).toContain('text-amber-400')
  })

  it('colour is NOT applied to the label element', () => {
    render(<AgentStatusCard label="Active" count={3} colour="text-emerald-400" />)
    const labelEl = screen.getByText('Active')
    expect(labelEl.className).not.toContain('text-emerald-400')
  })
})

describe('AgentStatusCard – multiple instances', () => {
  it('renders two cards with different labels and counts', () => {
    render(
      <>
        <AgentStatusCard label="Active" count={5} colour="text-emerald-400" />
        <AgentStatusCard label="Idle" count={3} colour="text-zinc-400" />
      </>
    )
    expect(screen.getByText('Active')).toBeInTheDocument()
    expect(screen.getByText('5')).toBeInTheDocument()
    expect(screen.getByText('Idle')).toBeInTheDocument()
    expect(screen.getByText('3')).toBeInTheDocument()
  })

  it('three cards with distinct counts all render correctly', () => {
    render(
      <>
        <AgentStatusCard label="A" count={1} colour="text-red-400" />
        <AgentStatusCard label="B" count={2} colour="text-blue-400" />
        <AgentStatusCard label="C" count={3} colour="text-green-400" />
      </>
    )
    expect(screen.getByText('1')).toBeInTheDocument()
    expect(screen.getByText('2')).toBeInTheDocument()
    expect(screen.getByText('3')).toBeInTheDocument()
  })

  it('zero count card alongside non-zero count card', () => {
    render(
      <>
        <AgentStatusCard label="Done" count={0} colour="text-zinc-500" />
        <AgentStatusCard label="Working" count={8} colour="text-blue-400" />
      </>
    )
    expect(screen.getByText('0')).toBeInTheDocument()
    expect(screen.getByText('8')).toBeInTheDocument()
  })
})

describe('AgentStatusCard – structure', () => {
  it('count element also has tabular-nums class', () => {
    render(<AgentStatusCard label="Active" count={5} colour="text-emerald-400" />)
    const countEl = screen.getByText('5')
    expect(countEl.className).toContain('tabular-nums')
  })

  it('count element has text-2xl class', () => {
    render(<AgentStatusCard label="Active" count={5} colour="text-emerald-400" />)
    const countEl = screen.getByText('5')
    expect(countEl.className).toContain('text-2xl')
  })
})
