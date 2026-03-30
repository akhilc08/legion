import { describe, it, expect } from 'vitest'
import { render, screen, fireEvent, within } from '@testing-library/react'
import { IssueRow } from './IssueRow'
import type { Issue, Agent, IssueStatus } from '@/lib/types'

const BASE_ISSUE: Issue = {
  id: 'issue-1',
  company_id: 'company-1',
  title: 'Fix login bug',
  description: 'Login fails with correct credentials',
  assignee_id: 'agent-1',
  parent_id: null,
  status: 'in_progress',
  output_path: null,
  attempt_count: 2,
  last_failure_reason: null,
  escalation_id: null,
  created_at: '2024-06-01T00:00:00Z',
  updated_at: '2024-06-01T00:00:00Z',
}

const BASE_AGENT: Agent = {
  id: 'agent-1',
  company_id: 'company-1',
  role: 'engineer',
  title: 'Jane Doe',
  system_prompt: '',
  manager_id: null,
  runtime: 'claude_code',
  status: 'working',
  monthly_budget: 100000,
  token_spend: 0,
  chat_token_spend: 0,
  pid: null,
  created_at: '2024-01-01T00:00:00Z',
  updated_at: '2024-01-01T00:00:00Z',
}

function renderInTable(issue: Issue, agents: Agent[] = [BASE_AGENT]) {
  return render(
    <table>
      <tbody>
        <IssueRow issue={issue} agents={agents} />
      </tbody>
    </table>
  )
}

describe('IssueRow – title', () => {
  it('renders issue title', () => {
    renderInTable(BASE_ISSUE)
    expect(screen.getByText('Fix login bug')).toBeInTheDocument()
  })

  it('renders a different title', () => {
    renderInTable({ ...BASE_ISSUE, title: 'Implement feature X' })
    expect(screen.getByText('Implement feature X')).toBeInTheDocument()
  })
})

describe('IssueRow – status badge', () => {
  it('renders in_progress status', () => {
    renderInTable({ ...BASE_ISSUE, status: 'in_progress' })
    expect(screen.getByText('in_progress')).toBeInTheDocument()
  })

  it('renders pending status', () => {
    renderInTable({ ...BASE_ISSUE, status: 'pending' })
    expect(screen.getByText('pending')).toBeInTheDocument()
  })

  it('renders blocked status', () => {
    renderInTable({ ...BASE_ISSUE, status: 'blocked' })
    expect(screen.getByText('blocked')).toBeInTheDocument()
  })

  it('renders done status', () => {
    renderInTable({ ...BASE_ISSUE, status: 'done' })
    expect(screen.getByText('done')).toBeInTheDocument()
  })

  it('renders failed status', () => {
    renderInTable({ ...BASE_ISSUE, status: 'failed' })
    expect(screen.getByText('failed')).toBeInTheDocument()
  })

  const statusColours: Record<IssueStatus, string> = {
    pending: 'text-zinc-400',
    in_progress: 'text-blue-400',
    blocked: 'text-amber-400',
    done: 'text-emerald-400',
    failed: 'text-red-400',
  }

  for (const [status, colour] of Object.entries(statusColours) as [IssueStatus, string][]) {
    it(`applies ${colour} for status "${status}"`, () => {
      renderInTable({ ...BASE_ISSUE, status })
      const badge = screen.getByText(status)
      expect(badge.className).toContain(colour)
    })
  }
})

describe('IssueRow – attempt_count', () => {
  it('renders attempt_count', () => {
    renderInTable({ ...BASE_ISSUE, attempt_count: 2 })
    expect(screen.getByText('2')).toBeInTheDocument()
  })

  it('renders attempt_count of 0', () => {
    renderInTable({ ...BASE_ISSUE, attempt_count: 0 })
    expect(screen.getByText('0')).toBeInTheDocument()
  })

  it('renders large attempt_count', () => {
    renderInTable({ ...BASE_ISSUE, attempt_count: 15 })
    expect(screen.getByText('15')).toBeInTheDocument()
  })
})

describe('IssueRow – assignee', () => {
  it('shows agent title when assignee_id matches an agent', () => {
    renderInTable(BASE_ISSUE, [BASE_AGENT])
    expect(screen.getByText('Jane Doe')).toBeInTheDocument()
  })

  it('shows "—" when assignee_id is null', () => {
    renderInTable({ ...BASE_ISSUE, assignee_id: null }, [BASE_AGENT])
    expect(screen.getByText('—')).toBeInTheDocument()
  })

  it('shows "—" when assignee_id does not match any agent', () => {
    renderInTable({ ...BASE_ISSUE, assignee_id: 'unknown-agent-id' }, [BASE_AGENT])
    expect(screen.getByText('—')).toBeInTheDocument()
  })

  it('shows "—" when agents array is empty', () => {
    renderInTable({ ...BASE_ISSUE, assignee_id: 'agent-1' }, [])
    expect(screen.getByText('—')).toBeInTheDocument()
  })

  it('shows correct agent from multiple agents', () => {
    const agents: Agent[] = [
      { ...BASE_AGENT, id: 'agent-1', title: 'Alice' },
      { ...BASE_AGENT, id: 'agent-2', title: 'Bob' },
    ]
    renderInTable({ ...BASE_ISSUE, assignee_id: 'agent-2' }, agents)
    expect(screen.getByText('Bob')).toBeInTheDocument()
    expect(screen.queryByText('Alice')).not.toBeInTheDocument()
  })
})

describe('IssueRow – expand/collapse', () => {
  it('clicking the row toggles expansion', () => {
    renderInTable({ ...BASE_ISSUE, description: 'Detailed description here' })
    expect(screen.queryByText('Detailed description here')).not.toBeInTheDocument()
    fireEvent.click(screen.getByText('Fix login bug').closest('tr')!)
    expect(screen.getByText('Detailed description here')).toBeInTheDocument()
  })

  it('clicking again collapses the row', () => {
    renderInTable({ ...BASE_ISSUE, description: 'Detail' })
    fireEvent.click(screen.getByText('Fix login bug').closest('tr')!)
    expect(screen.getByText('Detail')).toBeInTheDocument()
    fireEvent.click(screen.getByText('Fix login bug').closest('tr')!)
    expect(screen.queryByText('Detail')).not.toBeInTheDocument()
  })

  it('shows last_failure_reason when expanded', () => {
    renderInTable({ ...BASE_ISSUE, last_failure_reason: 'Connection refused' })
    fireEvent.click(screen.getByText('Fix login bug').closest('tr')!)
    expect(screen.getByText(/Connection refused/)).toBeInTheDocument()
  })

  it('failure reason is prefixed with "Failure:"', () => {
    renderInTable({ ...BASE_ISSUE, last_failure_reason: 'Timeout' })
    fireEvent.click(screen.getByText('Fix login bug').closest('tr')!)
    expect(screen.getByText(/Failure: Timeout/)).toBeInTheDocument()
  })

  it('does not show last_failure_reason when not expanded', () => {
    renderInTable({ ...BASE_ISSUE, last_failure_reason: 'Timeout' })
    expect(screen.queryByText(/Failure: Timeout/)).not.toBeInTheDocument()
  })

  it('shows output_path when expanded', () => {
    renderInTable({ ...BASE_ISSUE, output_path: '/workspace/output.txt' })
    fireEvent.click(screen.getByText('Fix login bug').closest('tr')!)
    expect(screen.getByText(/output\.txt/)).toBeInTheDocument()
  })

  it('does not show last_failure_reason row when null', () => {
    renderInTable({ ...BASE_ISSUE, last_failure_reason: null })
    fireEvent.click(screen.getByText('Fix login bug').closest('tr')!)
    expect(screen.queryByText(/Failure:/)).not.toBeInTheDocument()
  })
})
