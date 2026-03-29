import { it, expect, beforeAll, afterEach, afterAll } from 'vitest'
import { render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { MemoryRouter, Route, Routes } from 'react-router-dom'
import { server } from '@/test/server'
import { http, HttpResponse } from 'msw'
import { Hiring } from './index'
import type { PendingHire } from '@/lib/types'

beforeAll(() => server.listen())
afterEach(() => server.resetHandlers())
afterAll(() => server.close())

const mockHire: PendingHire = {
  id: 'h1', company_id: 'c1', requested_by_agent_id: 'a1',
  role_title: 'Senior Engineer', reporting_to_agent_id: 'a2',
  system_prompt: 'You are a senior engineer.', runtime: 'claude_code',
  budget_allocation: 50000, initial_task: 'Build auth module',
  status: 'pending', created_at: new Date().toISOString(),
}

it('renders pending hire and approve button', async () => {
  server.use(http.get('/api/companies/:id/hires', () => HttpResponse.json([mockHire])))
  const qc = new QueryClient({ defaultOptions: { queries: { retry: false } } })
  render(
    <QueryClientProvider client={qc}>
      <MemoryRouter initialEntries={['/companies/c1/hiring']}>
        <Routes><Route path="/companies/:companyId/hiring" element={<Hiring />} /></Routes>
      </MemoryRouter>
    </QueryClientProvider>
  )
  expect(await screen.findByText('Senior Engineer')).toBeInTheDocument()
  expect(screen.getByRole('button', { name: /approve/i })).toBeInTheDocument()
  expect(screen.getByRole('button', { name: /reject/i })).toBeInTheDocument()
})

it('calls approve endpoint on approve click', async () => {
  let approveHit = false
  server.use(
    http.get('/api/companies/:id/hires', () => HttpResponse.json([mockHire])),
    http.post('/api/companies/:id/hires/:hireId/approve', () => { approveHit = true; return HttpResponse.json({ status: 'approved' }) }),
  )
  const qc = new QueryClient({ defaultOptions: { queries: { retry: false } } })
  render(
    <QueryClientProvider client={qc}>
      <MemoryRouter initialEntries={['/companies/c1/hiring']}>
        <Routes><Route path="/companies/:companyId/hiring" element={<Hiring />} /></Routes>
      </MemoryRouter>
    </QueryClientProvider>
  )
  await userEvent.click(await screen.findByRole('button', { name: /approve/i }))
  expect(approveHit).toBe(true)
})
