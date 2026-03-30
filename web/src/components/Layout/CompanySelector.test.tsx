import { describe, it, expect, vi, beforeAll, afterEach, afterAll, beforeEach } from 'vitest'
import { render, screen, fireEvent } from '@testing-library/react'
import { MemoryRouter } from 'react-router-dom'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { http, HttpResponse } from 'msw'
import { server } from '@/test/server'
import type { Company } from '@/lib/types'

const mockNavigate = vi.fn()

vi.mock('react-router-dom', async () => {
  const actual = await vi.importActual<typeof import('react-router-dom')>('react-router-dom')
  return { ...actual, useNavigate: () => mockNavigate }
})

// Control companyId via module-level variable
let mockCompanyId: string | null = null

vi.mock('@/store/useAppStore', () => ({
  useAppStore: (sel: (s: { companyId: string | null }) => unknown) =>
    sel({ companyId: mockCompanyId }),
}))

import { CompanySelector } from './CompanySelector'

const COMPANIES: Company[] = [
  { id: 'c-1', name: 'Acme Corp', goal: 'Build rockets', created_at: '2024-01-01T00:00:00Z' },
  { id: 'c-2', name: 'Globex', goal: 'World domination', created_at: '2024-01-02T00:00:00Z' },
]

function makeQueryClient() {
  return new QueryClient({ defaultOptions: { queries: { retry: false } } })
}

function renderSelector() {
  return render(
    <QueryClientProvider client={makeQueryClient()}>
      <MemoryRouter>
        <CompanySelector />
      </MemoryRouter>
    </QueryClientProvider>
  )
}

beforeAll(() => server.listen({ onUnhandledRequest: 'bypass' }))
afterEach(() => {
  server.resetHandlers()
  mockNavigate.mockReset()
  mockCompanyId = null
})
afterAll(() => server.close())

describe('CompanySelector – no current company', () => {
  beforeEach(() => {
    server.use(http.get('/api/companies', () => HttpResponse.json(COMPANIES)))
    mockCompanyId = null
  })

  it('renders "Select company" when no company is selected', async () => {
    renderSelector()
    expect(await screen.findByText('Select company')).toBeInTheDocument()
  })

  it('renders a button', () => {
    renderSelector()
    expect(screen.getByRole('button')).toBeInTheDocument()
  })

  it('clicking the button calls navigate("/")', () => {
    renderSelector()
    fireEvent.click(screen.getByRole('button'))
    expect(mockNavigate).toHaveBeenCalledWith('/')
  })

  it('clicking the button calls navigate exactly once', () => {
    renderSelector()
    fireEvent.click(screen.getByRole('button'))
    expect(mockNavigate).toHaveBeenCalledOnce()
  })
})

describe('CompanySelector – with current company', () => {
  beforeEach(() => {
    server.use(http.get('/api/companies', () => HttpResponse.json(COMPANIES)))
    mockCompanyId = 'c-1'
  })

  it('renders the matching company name', async () => {
    renderSelector()
    expect(await screen.findByText('Acme Corp')).toBeInTheDocument()
  })

  it('does not show "Select company" when a company is matched', async () => {
    renderSelector()
    await screen.findByText('Acme Corp')
    expect(screen.queryByText('Select company')).not.toBeInTheDocument()
  })

  it('clicking the button navigates to "/"', async () => {
    renderSelector()
    await screen.findByText('Acme Corp')
    fireEvent.click(screen.getByRole('button'))
    expect(mockNavigate).toHaveBeenCalledWith('/')
  })
})

describe('CompanySelector – second company selected', () => {
  beforeEach(() => {
    server.use(http.get('/api/companies', () => HttpResponse.json(COMPANIES)))
    mockCompanyId = 'c-2'
  })

  it('renders the second company name', async () => {
    renderSelector()
    expect(await screen.findByText('Globex')).toBeInTheDocument()
  })
})

describe('CompanySelector – empty companies list', () => {
  beforeEach(() => {
    server.use(http.get('/api/companies', () => HttpResponse.json([])))
    mockCompanyId = 'c-1'
  })

  it('shows "Select company" when companies list is empty (no match)', async () => {
    renderSelector()
    // Query is loading then resolves with empty; companyId set but no match
    await screen.findByRole('button')
    expect(await screen.findByText('Select company')).toBeInTheDocument()
  })
})

describe('CompanySelector – unknown companyId', () => {
  beforeEach(() => {
    server.use(http.get('/api/companies', () => HttpResponse.json(COMPANIES)))
    mockCompanyId = 'unknown-id'
  })

  it('falls back to "Select company" when companyId does not match any company', async () => {
    renderSelector()
    await screen.findByRole('button')
    expect(await screen.findByText('Select company')).toBeInTheDocument()
  })
})
