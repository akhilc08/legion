import { it, expect, beforeAll, afterEach, afterAll } from 'vitest'
import { render, screen } from '@testing-library/react'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { MemoryRouter, Route, Routes } from 'react-router-dom'
import { server } from '@/test/server'
import { http, HttpResponse } from 'msw'
import { Issues } from './index'
import type { Issue } from '@/lib/types'

beforeAll(() => server.listen())
afterEach(() => server.resetHandlers())
afterAll(() => server.close())

const mockIssues: Partial<Issue>[] = [
  { id: 'i1', title: 'Fix login bug', status: 'in_progress', attempt_count: 1 },
  { id: 'i2', title: 'Deploy to staging', status: 'pending', attempt_count: 0 },
]

it('renders issue titles', async () => {
  server.use(
    http.get('/api/companies/:id/agents', () => HttpResponse.json([])),
    http.get('/api/companies/:id/issues', () => HttpResponse.json(mockIssues)),
  )
  const qc = new QueryClient({ defaultOptions: { queries: { retry: false } } })
  render(
    <QueryClientProvider client={qc}>
      <MemoryRouter initialEntries={['/companies/c1/issues']}>
        <Routes><Route path="/companies/:companyId/issues" element={<Issues />} /></Routes>
      </MemoryRouter>
    </QueryClientProvider>
  )
  expect(await screen.findByText('Fix login bug')).toBeInTheDocument()
  expect(await screen.findByText('Deploy to staging')).toBeInTheDocument()
})
