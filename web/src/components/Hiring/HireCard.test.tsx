import { describe, it, expect, vi } from 'vitest'
import { render, screen, fireEvent } from '@testing-library/react'
import { HireCard } from './HireCard'
import type { PendingHire } from '@/lib/types'

const BASE_HIRE: PendingHire = {
  id: 'hire-1',
  company_id: 'company-1',
  requested_by_agent_id: 'agent-requester',
  role_title: 'Backend Engineer',
  reporting_to_agent_id: 'agent-manager',
  system_prompt: 'You are a backend engineer.',
  runtime: 'claude_code',
  budget_allocation: 50000,
  initial_task: 'Set up database migrations',
  status: 'pending',
  created_at: '2024-06-15T10:00:00Z',
}

describe('HireCard – role_title', () => {
  it('renders role_title', () => {
    render(
      <HireCard
        hire={BASE_HIRE}
        onApprove={vi.fn()}
        onReject={vi.fn()}
        isApproving={false}
        isRejecting={false}
      />
    )
    expect(screen.getByText('Backend Engineer')).toBeInTheDocument()
  })

  it('renders a different role_title', () => {
    const hire: PendingHire = { ...BASE_HIRE, role_title: 'Frontend Designer' }
    render(
      <HireCard
        hire={hire}
        onApprove={vi.fn()}
        onReject={vi.fn()}
        isApproving={false}
        isRejecting={false}
      />
    )
    expect(screen.getByText('Frontend Designer')).toBeInTheDocument()
  })
})

describe('HireCard – runtime', () => {
  it('renders runtime', () => {
    render(
      <HireCard
        hire={BASE_HIRE}
        onApprove={vi.fn()}
        onReject={vi.fn()}
        isApproving={false}
        isRejecting={false}
      />
    )
    expect(screen.getByText('claude_code')).toBeInTheDocument()
  })

  it('renders openclaw runtime', () => {
    const hire: PendingHire = { ...BASE_HIRE, runtime: 'openclaw' }
    render(
      <HireCard
        hire={hire}
        onApprove={vi.fn()}
        onReject={vi.fn()}
        isApproving={false}
        isRejecting={false}
      />
    )
    expect(screen.getByText('openclaw')).toBeInTheDocument()
  })
})

describe('HireCard – budget_allocation', () => {
  it('renders budget_allocation with toLocaleString', () => {
    render(
      <HireCard
        hire={BASE_HIRE}
        onApprove={vi.fn()}
        onReject={vi.fn()}
        isApproving={false}
        isRejecting={false}
      />
    )
    const formatted = (50000).toLocaleString()
    expect(screen.getByText(new RegExp(formatted))).toBeInTheDocument()
  })

  it('renders "tokens" label alongside budget', () => {
    render(
      <HireCard
        hire={BASE_HIRE}
        onApprove={vi.fn()}
        onReject={vi.fn()}
        isApproving={false}
        isRejecting={false}
      />
    )
    expect(screen.getByText(/tokens/i)).toBeInTheDocument()
  })
})

describe('HireCard – initial_task', () => {
  it('shows initial_task when present', () => {
    render(
      <HireCard
        hire={BASE_HIRE}
        onApprove={vi.fn()}
        onReject={vi.fn()}
        isApproving={false}
        isRejecting={false}
      />
    )
    expect(screen.getByText('Set up database migrations')).toBeInTheDocument()
  })

  it('shows "Initial task" label when initial_task is present', () => {
    render(
      <HireCard
        hire={BASE_HIRE}
        onApprove={vi.fn()}
        onReject={vi.fn()}
        isApproving={false}
        isRejecting={false}
      />
    )
    expect(screen.getByText('Initial task')).toBeInTheDocument()
  })

  it('does not show initial_task section when initial_task is null', () => {
    const hire: PendingHire = { ...BASE_HIRE, initial_task: null }
    render(
      <HireCard
        hire={hire}
        onApprove={vi.fn()}
        onReject={vi.fn()}
        isApproving={false}
        isRejecting={false}
      />
    )
    expect(screen.queryByText('Initial task')).not.toBeInTheDocument()
  })

  it('does not show any task text when initial_task is null', () => {
    const hire: PendingHire = { ...BASE_HIRE, initial_task: null }
    render(
      <HireCard
        hire={hire}
        onApprove={vi.fn()}
        onReject={vi.fn()}
        isApproving={false}
        isRejecting={false}
      />
    )
    expect(screen.queryByText('Set up database migrations')).not.toBeInTheDocument()
  })
})

describe('HireCard – approve button', () => {
  it('renders Approve button', () => {
    render(
      <HireCard
        hire={BASE_HIRE}
        onApprove={vi.fn()}
        onReject={vi.fn()}
        isApproving={false}
        isRejecting={false}
      />
    )
    expect(screen.getByRole('button', { name: /approve/i })).toBeInTheDocument()
  })

  it('clicking Approve calls onApprove', () => {
    const onApprove = vi.fn()
    render(
      <HireCard
        hire={BASE_HIRE}
        onApprove={onApprove}
        onReject={vi.fn()}
        isApproving={false}
        isRejecting={false}
      />
    )
    fireEvent.click(screen.getByRole('button', { name: /approve/i }))
    expect(onApprove).toHaveBeenCalledOnce()
  })

  it('Approve button is not disabled when isApproving=false', () => {
    render(
      <HireCard
        hire={BASE_HIRE}
        onApprove={vi.fn()}
        onReject={vi.fn()}
        isApproving={false}
        isRejecting={false}
      />
    )
    expect(screen.getByRole('button', { name: /approve/i })).not.toBeDisabled()
  })

  it('Approve button is disabled when isApproving=true', () => {
    render(
      <HireCard
        hire={BASE_HIRE}
        onApprove={vi.fn()}
        onReject={vi.fn()}
        isApproving={true}
        isRejecting={false}
      />
    )
    expect(screen.getByRole('button', { name: /approving/i })).toBeDisabled()
  })

  it('shows "Approving…" text when isApproving=true', () => {
    render(
      <HireCard
        hire={BASE_HIRE}
        onApprove={vi.fn()}
        onReject={vi.fn()}
        isApproving={true}
        isRejecting={false}
      />
    )
    expect(screen.getByText(/approving…/i)).toBeInTheDocument()
  })
})

describe('HireCard – reject button', () => {
  it('renders Reject button', () => {
    render(
      <HireCard
        hire={BASE_HIRE}
        onApprove={vi.fn()}
        onReject={vi.fn()}
        isApproving={false}
        isRejecting={false}
      />
    )
    expect(screen.getByRole('button', { name: /^reject$/i })).toBeInTheDocument()
  })

  it('clicking Reject calls onReject', () => {
    const onReject = vi.fn()
    render(
      <HireCard
        hire={BASE_HIRE}
        onApprove={vi.fn()}
        onReject={onReject}
        isApproving={false}
        isRejecting={false}
      />
    )
    fireEvent.click(screen.getByRole('button', { name: /^reject$/i }))
    expect(onReject).toHaveBeenCalledOnce()
  })

  it('Reject button is not disabled when isRejecting=false', () => {
    render(
      <HireCard
        hire={BASE_HIRE}
        onApprove={vi.fn()}
        onReject={vi.fn()}
        isApproving={false}
        isRejecting={false}
      />
    )
    expect(screen.getByRole('button', { name: /^reject$/i })).not.toBeDisabled()
  })

  it('Reject button is disabled when isRejecting=true', () => {
    render(
      <HireCard
        hire={BASE_HIRE}
        onApprove={vi.fn()}
        onReject={vi.fn()}
        isApproving={false}
        isRejecting={true}
      />
    )
    expect(screen.getByRole('button', { name: /rejecting/i })).toBeDisabled()
  })

  it('shows "Rejecting…" text when isRejecting=true', () => {
    render(
      <HireCard
        hire={BASE_HIRE}
        onApprove={vi.fn()}
        onReject={vi.fn()}
        isApproving={false}
        isRejecting={true}
      />
    )
    expect(screen.getByText(/rejecting…/i)).toBeInTheDocument()
  })
})

describe('HireCard – both loading states simultaneously', () => {
  it('isApproving=true does not disable the Reject button', () => {
    render(
      <HireCard
        hire={BASE_HIRE}
        onApprove={vi.fn()}
        onReject={vi.fn()}
        isApproving={true}
        isRejecting={false}
      />
    )
    expect(screen.getByRole('button', { name: /^reject$/i })).not.toBeDisabled()
  })

  it('isRejecting=true does not disable the Approve button', () => {
    render(
      <HireCard
        hire={BASE_HIRE}
        onApprove={vi.fn()}
        onReject={vi.fn()}
        isApproving={false}
        isRejecting={true}
      />
    )
    expect(screen.getByRole('button', { name: /approve/i })).not.toBeDisabled()
  })
})
