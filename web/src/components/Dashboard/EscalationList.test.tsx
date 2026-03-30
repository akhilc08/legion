import { describe, it, expect, vi } from 'vitest'
import { render, screen, fireEvent } from '@testing-library/react'
import { EscalationList } from './EscalationList'
import type { Notification } from '@/lib/types'

const BASE_NOTIFICATION: Notification = {
  id: 'notif-1',
  company_id: 'company-1',
  type: 'escalation_triggered',
  escalation_id: 'esc-1',
  payload: {},
  dismissed_at: null,
  created_at: '2024-06-15T10:00:00Z',
}

describe('EscalationList – empty state', () => {
  it('shows "No active notifications." when list is empty', () => {
    render(<EscalationList notifications={[]} onDismiss={vi.fn()} />)
    expect(screen.getByText('No active notifications.')).toBeInTheDocument()
  })

  it('does not render a list element when empty', () => {
    render(<EscalationList notifications={[]} onDismiss={vi.fn()} />)
    expect(screen.queryByRole('list')).not.toBeInTheDocument()
  })
})

describe('EscalationList – rendering notifications', () => {
  it('renders a single notification type', () => {
    render(
      <EscalationList
        notifications={[BASE_NOTIFICATION]}
        onDismiss={vi.fn()}
      />
    )
    expect(screen.getByText('escalation_triggered')).toBeInTheDocument()
  })

  it('renders a dismiss button for each notification', () => {
    render(
      <EscalationList
        notifications={[BASE_NOTIFICATION]}
        onDismiss={vi.fn()}
      />
    )
    expect(screen.getByRole('button', { name: /dismiss/i })).toBeInTheDocument()
  })

  it('renders multiple notifications', () => {
    const notifications: Notification[] = [
      { ...BASE_NOTIFICATION, id: 'n-1', type: 'type_one' },
      { ...BASE_NOTIFICATION, id: 'n-2', type: 'type_two' },
      { ...BASE_NOTIFICATION, id: 'n-3', type: 'type_three' },
    ]
    render(<EscalationList notifications={notifications} onDismiss={vi.fn()} />)
    expect(screen.getByText('type_one')).toBeInTheDocument()
    expect(screen.getByText('type_two')).toBeInTheDocument()
    expect(screen.getByText('type_three')).toBeInTheDocument()
  })

  it('renders one dismiss button per notification', () => {
    const notifications: Notification[] = [
      { ...BASE_NOTIFICATION, id: 'n-1', type: 'type_one' },
      { ...BASE_NOTIFICATION, id: 'n-2', type: 'type_two' },
    ]
    render(<EscalationList notifications={notifications} onDismiss={vi.fn()} />)
    expect(screen.getAllByRole('button', { name: /dismiss/i })).toHaveLength(2)
  })

  it('renders formatted timestamp for each notification', () => {
    render(
      <EscalationList
        notifications={[BASE_NOTIFICATION]}
        onDismiss={vi.fn()}
      />
    )
    const expected = new Date(BASE_NOTIFICATION.created_at).toLocaleString()
    expect(screen.getByText(expected)).toBeInTheDocument()
  })

  it('does not show "No active notifications." when notifications exist', () => {
    render(
      <EscalationList
        notifications={[BASE_NOTIFICATION]}
        onDismiss={vi.fn()}
      />
    )
    expect(screen.queryByText('No active notifications.')).not.toBeInTheDocument()
  })

  it('renders a list (ul) when notifications are present', () => {
    render(
      <EscalationList
        notifications={[BASE_NOTIFICATION]}
        onDismiss={vi.fn()}
      />
    )
    expect(screen.getByRole('list')).toBeInTheDocument()
  })
})

describe('EscalationList – dismiss behaviour', () => {
  it('calls onDismiss with correct id when dismiss clicked', () => {
    const onDismiss = vi.fn()
    render(
      <EscalationList
        notifications={[BASE_NOTIFICATION]}
        onDismiss={onDismiss}
      />
    )
    fireEvent.click(screen.getByRole('button', { name: /dismiss/i }))
    expect(onDismiss).toHaveBeenCalledWith('notif-1')
  })

  it('calls onDismiss exactly once per click', () => {
    const onDismiss = vi.fn()
    render(
      <EscalationList
        notifications={[BASE_NOTIFICATION]}
        onDismiss={onDismiss}
      />
    )
    fireEvent.click(screen.getByRole('button', { name: /dismiss/i }))
    expect(onDismiss).toHaveBeenCalledOnce()
  })

  it('calls onDismiss with correct id for second notification', () => {
    const onDismiss = vi.fn()
    const notifications: Notification[] = [
      { ...BASE_NOTIFICATION, id: 'n-1', type: 'first' },
      { ...BASE_NOTIFICATION, id: 'n-2', type: 'second' },
    ]
    render(<EscalationList notifications={notifications} onDismiss={onDismiss} />)
    const buttons = screen.getAllByRole('button', { name: /dismiss/i })
    fireEvent.click(buttons[1])
    expect(onDismiss).toHaveBeenCalledWith('n-2')
  })

  it('does not call onDismiss until button clicked', () => {
    const onDismiss = vi.fn()
    render(
      <EscalationList
        notifications={[BASE_NOTIFICATION]}
        onDismiss={onDismiss}
      />
    )
    expect(onDismiss).not.toHaveBeenCalled()
  })

  it('clicking first dismiss button does not fire with second id', () => {
    const onDismiss = vi.fn()
    const notifications: Notification[] = [
      { ...BASE_NOTIFICATION, id: 'n-1', type: 'first' },
      { ...BASE_NOTIFICATION, id: 'n-2', type: 'second' },
    ]
    render(<EscalationList notifications={notifications} onDismiss={onDismiss} />)
    const buttons = screen.getAllByRole('button', { name: /dismiss/i })
    fireEvent.click(buttons[0])
    expect(onDismiss).toHaveBeenCalledWith('n-1')
    expect(onDismiss).not.toHaveBeenCalledWith('n-2')
  })
})

describe('EscalationList – notification types', () => {
  it('renders hire_pending type', () => {
    const n: Notification = { ...BASE_NOTIFICATION, type: 'hire_pending' }
    render(<EscalationList notifications={[n]} onDismiss={vi.fn()} />)
    expect(screen.getByText('hire_pending')).toBeInTheDocument()
  })

  it('renders issue_blocked type', () => {
    const n: Notification = { ...BASE_NOTIFICATION, type: 'issue_blocked' }
    render(<EscalationList notifications={[n]} onDismiss={vi.fn()} />)
    expect(screen.getByText('issue_blocked')).toBeInTheDocument()
  })
})
