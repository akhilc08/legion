import { describe, it, expect, beforeAll, afterAll, afterEach } from 'vitest'
import { render, screen, fireEvent, waitFor } from '@testing-library/react'
import { MemoryRouter, Route, Routes } from 'react-router-dom'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { http, HttpResponse } from 'msw'
import { server } from '@/test/server'
import { Audit } from './index'
import type { AuditLog } from '@/lib/types'

function makeQueryClient() {
  return new QueryClient({ defaultOptions: { queries: { retry: false } } })
}

function renderAudit(companyId = 'company-1', qc?: QueryClient) {
  const client = qc ?? makeQueryClient()
  return render(
    <QueryClientProvider client={client}>
      <MemoryRouter initialEntries={[`/companies/${companyId}/audit`]}>
        <Routes>
          <Route path="/companies/:companyId/audit" element={<Audit />} />
        </Routes>
      </MemoryRouter>
    </QueryClientProvider>
  )
}

const BASE_LOG: AuditLog = {
  id: 'log-1',
  company_id: 'company-1',
  actor_id: 'abcdefgh-uuid-1234',
  event_type: 'agent.started',
  payload: { key: 'value', count: 42 },
  created_at: '2024-06-15T10:30:00Z',
}

beforeAll(() => server.listen({ onUnhandledRequest: 'error' }))
afterEach(() => server.resetHandlers())
afterAll(() => server.close())

describe('Audit – heading and empty state', () => {
  it('renders "Audit Log" heading', async () => {
    server.use(
      http.get('/api/companies/company-1/audit', () => HttpResponse.json([]))
    )
    renderAudit()
    await screen.findByText('Audit Log')
  })

  it('renders table column headers', async () => {
    server.use(
      http.get('/api/companies/company-1/audit', () => HttpResponse.json([]))
    )
    renderAudit()
    await screen.findByText('Audit Log')
    expect(screen.getByText('Time')).toBeInTheDocument()
    expect(screen.getByText('Event')).toBeInTheDocument()
    expect(screen.getByText('Actor')).toBeInTheDocument()
    expect(screen.getByText('Payload')).toBeInTheDocument()
  })

  it('shows "No events yet." when list is empty', async () => {
    server.use(
      http.get('/api/companies/company-1/audit', () => HttpResponse.json([]))
    )
    renderAudit()
    await screen.findByText('No events yet.')
  })

  it('does not show "No events yet." when there are entries', async () => {
    server.use(
      http.get('/api/companies/company-1/audit', () => HttpResponse.json([BASE_LOG]))
    )
    renderAudit()
    await screen.findByText('agent.started')
    expect(screen.queryByText('No events yet.')).not.toBeInTheDocument()
  })
})

describe('Audit – log entry rendering', () => {
  it('renders event_type', async () => {
    server.use(
      http.get('/api/companies/company-1/audit', () => HttpResponse.json([BASE_LOG]))
    )
    renderAudit()
    await screen.findByText('agent.started')
  })

  it('renders actor_id truncated to first 8 chars + ellipsis', async () => {
    server.use(
      http.get('/api/companies/company-1/audit', () => HttpResponse.json([BASE_LOG]))
    )
    renderAudit()
    await screen.findByText('abcdefgh\u2026')
  })

  it('renders "—" when actor_id is null', async () => {
    const log: AuditLog = { ...BASE_LOG, actor_id: null }
    server.use(
      http.get('/api/companies/company-1/audit', () => HttpResponse.json([log]))
    )
    renderAudit()
    await screen.findByText('agent.started')
    expect(screen.getByText('—')).toBeInTheDocument()
  })

  it('does not render full actor_id UUID', async () => {
    server.use(
      http.get('/api/companies/company-1/audit', () => HttpResponse.json([BASE_LOG]))
    )
    renderAudit()
    await screen.findByText('agent.started')
    expect(screen.queryByText('abcdefgh-uuid-1234')).not.toBeInTheDocument()
  })

  it('renders formatted timestamp using toLocaleString', async () => {
    server.use(
      http.get('/api/companies/company-1/audit', () => HttpResponse.json([BASE_LOG]))
    )
    renderAudit()
    await screen.findByText('agent.started')
    const expected = new Date('2024-06-15T10:30:00Z').toLocaleString()
    expect(screen.getByText(expected)).toBeInTheDocument()
  })

  it('renders multiple log entries', async () => {
    const logs: AuditLog[] = [
      { ...BASE_LOG, id: 'log-1', event_type: 'agent.started' },
      { ...BASE_LOG, id: 'log-2', event_type: 'issue.created' },
      { ...BASE_LOG, id: 'log-3', event_type: 'hire.approved' },
    ]
    server.use(
      http.get('/api/companies/company-1/audit', () => HttpResponse.json(logs))
    )
    renderAudit()
    await screen.findByText('agent.started')
    expect(screen.getByText('issue.created')).toBeInTheDocument()
    expect(screen.getByText('hire.approved')).toBeInTheDocument()
  })

  it('shows "▼ show" text for collapsed rows', async () => {
    server.use(
      http.get('/api/companies/company-1/audit', () => HttpResponse.json([BASE_LOG]))
    )
    renderAudit()
    await screen.findByText('▼ show')
  })
})

describe('Audit – expand/collapse', () => {
  it('clicking a row shows "▲ hide" text', async () => {
    server.use(
      http.get('/api/companies/company-1/audit', () => HttpResponse.json([BASE_LOG]))
    )
    renderAudit()
    await screen.findByText('▼ show')
    fireEvent.click(screen.getByText('agent.started').closest('tr')!)
    await screen.findByText('▲ hide')
  })

  it('clicking a row reveals JSON-formatted payload', async () => {
    server.use(
      http.get('/api/companies/company-1/audit', () => HttpResponse.json([BASE_LOG]))
    )
    renderAudit()
    await screen.findByText('▼ show')
    fireEvent.click(screen.getByText('agent.started').closest('tr')!)
    await waitFor(() => {
      const pre = document.querySelector('pre')
      expect(pre).toBeInTheDocument()
      expect(pre!.textContent).toContain('"key"')
      expect(pre!.textContent).toContain('"value"')
    })
  })

  it('payload is JSON.stringify with 2-space indent', async () => {
    server.use(
      http.get('/api/companies/company-1/audit', () => HttpResponse.json([BASE_LOG]))
    )
    renderAudit()
    await screen.findByText('▼ show')
    fireEvent.click(screen.getByText('agent.started').closest('tr')!)
    const expected = JSON.stringify(BASE_LOG.payload, null, 2)
    await waitFor(() => {
      const pre = document.querySelector('pre')
      expect(pre!.textContent).toBe(expected)
    })
  })

  it('clicking an expanded row collapses it back', async () => {
    server.use(
      http.get('/api/companies/company-1/audit', () => HttpResponse.json([BASE_LOG]))
    )
    renderAudit()
    await screen.findByText('▼ show')
    fireEvent.click(screen.getByText('agent.started').closest('tr')!)
    await screen.findByText('▲ hide')
    fireEvent.click(screen.getByText('agent.started').closest('tr')!)
    await screen.findByText('▼ show')
    expect(screen.queryByText('▲ hide')).not.toBeInTheDocument()
  })

  it('collapsing removes the payload pre element', async () => {
    server.use(
      http.get('/api/companies/company-1/audit', () => HttpResponse.json([BASE_LOG]))
    )
    renderAudit()
    await screen.findByText('▼ show')
    fireEvent.click(screen.getByText('agent.started').closest('tr')!)
    await waitFor(() => expect(document.querySelector('pre')).toBeInTheDocument())
    fireEvent.click(screen.getByText('agent.started').closest('tr')!)
    await waitFor(() => expect(document.querySelector('pre')).not.toBeInTheDocument())
  })

  it('two rows expand and collapse independently', async () => {
    const logs: AuditLog[] = [
      { ...BASE_LOG, id: 'log-1', event_type: 'event.one', payload: { a: 1 } },
      { ...BASE_LOG, id: 'log-2', event_type: 'event.two', payload: { b: 2 } },
    ]
    server.use(
      http.get('/api/companies/company-1/audit', () => HttpResponse.json(logs))
    )
    renderAudit()
    await screen.findByText('event.one')

    // Expand row one only
    fireEvent.click(screen.getByText('event.one').closest('tr')!)
    await waitFor(() => expect(document.querySelectorAll('pre')).toHaveLength(1))
    expect(document.querySelector('pre')!.textContent).toContain('"a"')

    // Expand row two as well — only one can be expanded at a time (single expanded state)
    fireEvent.click(screen.getByText('event.two').closest('tr')!)
    await waitFor(() => {
      expect(document.querySelector('pre')!.textContent).toContain('"b"')
    })
  })

  it('expanding row two collapses row one', async () => {
    const logs: AuditLog[] = [
      { ...BASE_LOG, id: 'log-1', event_type: 'event.one', payload: { a: 1 } },
      { ...BASE_LOG, id: 'log-2', event_type: 'event.two', payload: { b: 2 } },
    ]
    server.use(
      http.get('/api/companies/company-1/audit', () => HttpResponse.json(logs))
    )
    renderAudit()
    await screen.findByText('event.one')

    fireEvent.click(screen.getByText('event.one').closest('tr')!)
    await waitFor(() => expect(document.querySelector('pre')).toBeInTheDocument())

    fireEvent.click(screen.getByText('event.two').closest('tr')!)
    await waitFor(() => {
      expect(document.querySelector('pre')!.textContent).toContain('"b"')
      expect(document.querySelector('pre')!.textContent).not.toContain('"a"')
    })
  })
})

describe('Audit – API call', () => {
  it('requests the audit endpoint with ?limit=100', async () => {
    const spy = vi.fn(() => HttpResponse.json([]))
    server.use(
      http.get('/api/companies/company-1/audit', ({ request }) => {
        const url = new URL(request.url)
        if (url.searchParams.get('limit') === '100') {
          spy()
        }
        return HttpResponse.json([])
      })
    )
    renderAudit()
    await screen.findByText('No events yet.')
    expect(spy).toHaveBeenCalled()
  })
})

describe('Audit – null payload', () => {
  it('renders entry with empty payload object gracefully', async () => {
    const log: AuditLog = { ...BASE_LOG, payload: {} }
    server.use(
      http.get('/api/companies/company-1/audit', () => HttpResponse.json([log]))
    )
    renderAudit()
    await screen.findByText('agent.started')
    fireEvent.click(screen.getByText('agent.started').closest('tr')!)
    await waitFor(() => {
      const pre = document.querySelector('pre')
      expect(pre!.textContent).toBe('{}')
    })
  })
})
