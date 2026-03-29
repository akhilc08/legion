import { describe, it, expect, beforeAll, afterEach, afterAll } from 'vitest'
import { render, screen, waitFor } from '@testing-library/react'
import { http, HttpResponse } from 'msw'
import { server } from '@/test/server'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { MemoryRouter, Route, Routes } from 'react-router-dom'
import { FSBrowser } from './index'

function wrapper(companyId = 'company-1') {
  const qc = new QueryClient({ defaultOptions: { queries: { retry: false } } })
  return (
    <QueryClientProvider client={qc}>
      <MemoryRouter initialEntries={[`/companies/${companyId}/files`]}>
        <Routes>
          <Route path="/companies/:companyId/files" element={<FSBrowser />} />
        </Routes>
      </MemoryRouter>
    </QueryClientProvider>
  )
}

beforeAll(() => server.listen())
afterEach(() => server.resetHandlers())
afterAll(() => server.close())

describe('FSBrowser', () => {
  it('renders file entries from browse endpoint', async () => {
    server.use(
      http.get('/api/companies/company-1/fs/browse', () =>
        HttpResponse.json([
          { name: 'reports', is_dir: true, size: 0 },
          { name: 'README.md', is_dir: false, size: 1024 },
        ])
      ),
      http.get('/api/companies/company-1/fs/permissions', () => HttpResponse.json([])),
      http.get('/api/companies/company-1/agents', () => HttpResponse.json([]))
    )

    render(wrapper())

    await waitFor(() => {
      expect(screen.getByText('reports')).toBeInTheDocument()
      expect(screen.getByText('README.md')).toBeInTheDocument()
    })
  })

  it('renders permissions table with a permission row', async () => {
    server.use(
      http.get('/api/companies/company-1/fs/browse', () => HttpResponse.json([])),
      http.get('/api/companies/company-1/fs/permissions', () =>
        HttpResponse.json([
          {
            id: 'perm-1',
            agent_id: 'agent-1',
            path: '/reports',
            permission_level: 'read',
            granted_by: null,
          },
        ])
      ),
      http.get('/api/companies/company-1/agents', () =>
        HttpResponse.json([
          {
            id: 'agent-1',
            company_id: 'company-1',
            role: 'analyst',
            title: 'Data Analyst',
            system_prompt: '',
            manager_id: null,
            runtime: 'claude_code',
            status: 'idle',
            monthly_budget: 100000,
            token_spend: 0,
            chat_token_spend: 0,
            pid: null,
            created_at: new Date().toISOString(),
            updated_at: new Date().toISOString(),
          },
        ])
      )
    )

    render(wrapper())

    await waitFor(() => {
      expect(screen.getByText('/reports')).toBeInTheDocument()
      expect(screen.getAllByText('read').length).toBeGreaterThan(0)
      expect(screen.getByText('Revoke')).toBeInTheDocument()
    })
  })
})
