import { describe, it, expect, beforeAll, afterEach, afterAll } from 'vitest'
import { render, screen } from '@testing-library/react'
import { QueryClientProvider } from '@tanstack/react-query'
import { QueryClient } from '@tanstack/react-query'
import { MemoryRouter, Route, Routes } from 'react-router-dom'
import { server } from '@/test/server'
import { http, HttpResponse } from 'msw'
import { Dashboard } from './index'
import type { Agent, Notification } from '@/lib/types'

beforeAll(() => server.listen())
afterEach(() => server.resetHandlers())
afterAll(() => server.close())

function makeClient() {
  return new QueryClient({ defaultOptions: { queries: { retry: false } } })
}

function wrapper(companyId: string) {
  return function W({ children }: { children: React.ReactNode }) {
    return (
      <QueryClientProvider client={makeClient()}>
        <MemoryRouter initialEntries={[`/companies/${companyId}/dashboard`]}>
          <Routes>
            <Route path="/companies/:companyId/dashboard" element={<>{children}</>} />
          </Routes>
        </MemoryRouter>
      </QueryClientProvider>
    )
  }
}

it('shows working agent count', async () => {
  const agents: Partial<Agent>[] = [
    { id: '1', status: 'working' },
    { id: '2', status: 'idle' },
    { id: '3', status: 'working' },
  ]
  server.use(
    http.get('/api/companies/:id/agents', () => HttpResponse.json(agents)),
    http.get('/api/companies/:id/notifications', () => HttpResponse.json([])),
  )
  const { findByText } = render(<Dashboard />, { wrapper: wrapper('c-1') })
  expect(await findByText('2')).toBeInTheDocument() // 2 working agents
})
