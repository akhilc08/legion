import { describe, it, expect, vi, beforeAll, afterAll, afterEach } from 'vitest'
import { render, screen, fireEvent, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { MemoryRouter } from 'react-router-dom'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { http, HttpResponse } from 'msw'
import { server } from '@/test/server'
import { CompanyList } from './CompanyList'
import { useAppStore } from '@/store/useAppStore'
import type { Company } from '@/lib/types'

const mockNavigate = vi.fn()

vi.mock('react-router-dom', async () => {
  const actual = await vi.importActual<typeof import('react-router-dom')>('react-router-dom')
  return { ...actual, useNavigate: () => mockNavigate }
})

function makeQueryClient() {
  return new QueryClient({
    defaultOptions: { queries: { retry: false } },
  })
}

function renderCompanyList(qc?: QueryClient) {
  const client = qc ?? makeQueryClient()
  return render(
    <QueryClientProvider client={client}>
      <MemoryRouter>
        <CompanyList />
      </MemoryRouter>
    </QueryClientProvider>
  )
}

const COMPANIES: Company[] = [
  { id: 'c-1', name: 'Acme Corp', goal: 'Build rockets', created_at: '2024-01-01T00:00:00Z' },
  { id: 'c-2', name: 'Globex', goal: 'World domination', created_at: '2024-01-02T00:00:00Z' },
]

beforeAll(() => server.listen({ onUnhandledRequest: 'error' }))
afterEach(() => {
  server.resetHandlers()
  mockNavigate.mockReset()
  localStorage.clear()
  useAppStore.setState({ token: null, companyId: null, agentId: null })
})
afterAll(() => server.close())

describe('CompanyList – heading', () => {
  it('renders "Your companies" heading', async () => {
    server.use(http.get('/api/companies', () => HttpResponse.json([])))
    renderCompanyList()
    await screen.findByText('Your companies')
  })

  it('renders a "Sign out" button', async () => {
    server.use(http.get('/api/companies', () => HttpResponse.json([])))
    renderCompanyList()
    await screen.findByRole('button', { name: /sign out/i })
  })
})

describe('CompanyList – empty state', () => {
  it('shows no company items when list is empty', async () => {
    server.use(http.get('/api/companies', () => HttpResponse.json([])))
    renderCompanyList()
    await screen.findByText('Your companies')
    expect(screen.queryAllByRole('button').filter(b => b.textContent?.includes('Acme'))).toHaveLength(0)
  })

  it('shows loading text while fetching', () => {
    server.use(
      http.get('/api/companies', () =>
        new Promise(() => {}) // never resolves
      )
    )
    renderCompanyList()
    expect(screen.getByText(/loading/i)).toBeInTheDocument()
  })
})

describe('CompanyList – list rendering', () => {
  it('renders each company name', async () => {
    server.use(http.get('/api/companies', () => HttpResponse.json(COMPANIES)))
    renderCompanyList()
    await screen.findByText('Acme Corp')
    expect(screen.getByText('Globex')).toBeInTheDocument()
  })

  it('renders each company goal', async () => {
    server.use(http.get('/api/companies', () => HttpResponse.json(COMPANIES)))
    renderCompanyList()
    await screen.findByText('Build rockets')
    expect(screen.getByText('World domination')).toBeInTheDocument()
  })

  it('renders correct number of company buttons', async () => {
    server.use(http.get('/api/companies', () => HttpResponse.json(COMPANIES)))
    renderCompanyList()
    await screen.findByText('Acme Corp')
    // Two company buttons plus Sign out, Create buttons
    const acmeBtn = screen.getByText('Acme Corp').closest('button')
    const globexBtn = screen.getByText('Globex').closest('button')
    expect(acmeBtn).toBeInTheDocument()
    expect(globexBtn).toBeInTheDocument()
  })

  it('clicking a company navigates to its dashboard', async () => {
    server.use(http.get('/api/companies', () => HttpResponse.json(COMPANIES)))
    renderCompanyList()
    await screen.findByText('Acme Corp')
    fireEvent.click(screen.getByText('Acme Corp').closest('button')!)
    expect(mockNavigate).toHaveBeenCalledWith('/companies/c-1/dashboard')
  })

  it('clicking second company navigates to its dashboard', async () => {
    server.use(http.get('/api/companies', () => HttpResponse.json(COMPANIES)))
    renderCompanyList()
    await screen.findByText('Globex')
    fireEvent.click(screen.getByText('Globex').closest('button')!)
    expect(mockNavigate).toHaveBeenCalledWith('/companies/c-2/dashboard')
  })
})

describe('CompanyList – create form', () => {
  it('renders "New company" section heading', async () => {
    server.use(http.get('/api/companies', () => HttpResponse.json([])))
    renderCompanyList()
    await screen.findByText('New company')
  })

  it('renders Name input', async () => {
    server.use(http.get('/api/companies', () => HttpResponse.json([])))
    renderCompanyList()
    await screen.findByPlaceholderText('Name')
  })

  it('renders Goal input', async () => {
    server.use(http.get('/api/companies', () => HttpResponse.json([])))
    renderCompanyList()
    await screen.findByPlaceholderText('Goal')
  })

  it('renders a "Create" submit button', async () => {
    server.use(http.get('/api/companies', () => HttpResponse.json([])))
    renderCompanyList()
    await screen.findByRole('button', { name: /^create$/i })
  })

  it('typing into name field updates value', async () => {
    server.use(http.get('/api/companies', () => HttpResponse.json([])))
    renderCompanyList()
    await screen.findByPlaceholderText('Name')
    const nameInput = screen.getByPlaceholderText('Name')
    await userEvent.type(nameInput, 'My New Co')
    expect(nameInput).toHaveValue('My New Co')
  })

  it('typing into goal field updates value', async () => {
    server.use(http.get('/api/companies', () => HttpResponse.json([])))
    renderCompanyList()
    await screen.findByPlaceholderText('Goal')
    const goalInput = screen.getByPlaceholderText('Goal')
    await userEvent.type(goalInput, 'Achieve greatness')
    expect(goalInput).toHaveValue('Achieve greatness')
  })

  it('name input has required attribute', async () => {
    server.use(http.get('/api/companies', () => HttpResponse.json([])))
    renderCompanyList()
    await screen.findByPlaceholderText('Name')
    expect(screen.getByPlaceholderText('Name')).toBeRequired()
  })
})

describe('CompanyList – create submission', () => {
  it('calls POST /api/companies with name and goal', async () => {
    const spy = vi.fn(() =>
      HttpResponse.json({ id: 'c-new', name: 'My New Co', goal: 'Achieve greatness', created_at: new Date().toISOString() })
    )
    server.use(
      http.get('/api/companies', () => HttpResponse.json([])),
      http.post('/api/companies', spy),
    )
    renderCompanyList()
    await screen.findByPlaceholderText('Name')
    await userEvent.type(screen.getByPlaceholderText('Name'), 'My New Co')
    await userEvent.type(screen.getByPlaceholderText('Goal'), 'Achieve greatness')
    fireEvent.submit(screen.getByRole('button', { name: /^create$/i }).closest('form')!)
    await waitFor(() => expect(spy).toHaveBeenCalledOnce())
  })

  it('navigates to new company dashboard on success', async () => {
    const newCompany: Company = {
      id: 'c-new',
      name: 'My New Co',
      goal: 'Achieve greatness',
      created_at: new Date().toISOString(),
    }
    server.use(
      http.get('/api/companies', () => HttpResponse.json([])),
      http.post('/api/companies', () => HttpResponse.json(newCompany)),
    )
    renderCompanyList()
    await screen.findByPlaceholderText('Name')
    await userEvent.type(screen.getByPlaceholderText('Name'), 'My New Co')
    await userEvent.type(screen.getByPlaceholderText('Goal'), 'Achieve greatness')
    fireEvent.submit(screen.getByRole('button', { name: /^create$/i }).closest('form')!)
    await waitFor(() =>
      expect(mockNavigate).toHaveBeenCalledWith('/companies/c-new/dashboard')
    )
  })

  it('shows "Creating…" text during pending mutation', async () => {
    let resolve!: (value: unknown) => void
    server.use(
      http.get('/api/companies', () => HttpResponse.json([])),
      http.post('/api/companies', () => new Promise((r) => { resolve = r })),
    )
    renderCompanyList()
    await screen.findByPlaceholderText('Name')
    await userEvent.type(screen.getByPlaceholderText('Name'), 'Co')
    fireEvent.submit(screen.getByRole('button', { name: /^create$/i }).closest('form')!)
    await screen.findByText(/creating…/i)
    resolve(HttpResponse.json({ id: 'c-x', name: 'Co', goal: '', created_at: new Date().toISOString() }))
  })

  it('disables Create button while pending', async () => {
    let resolve!: (value: unknown) => void
    server.use(
      http.get('/api/companies', () => HttpResponse.json([])),
      http.post('/api/companies', () => new Promise((r) => { resolve = r })),
    )
    renderCompanyList()
    await screen.findByPlaceholderText('Name')
    await userEvent.type(screen.getByPlaceholderText('Name'), 'Co')
    fireEvent.submit(screen.getByRole('button', { name: /^create$/i }).closest('form')!)
    await screen.findByText(/creating…/i)
    expect(screen.getByRole('button', { name: /creating…/i })).toBeDisabled()
    resolve(HttpResponse.json({ id: 'c-x', name: 'Co', goal: '', created_at: new Date().toISOString() }))
  })
})

describe('CompanyList – sign out', () => {
  it('clicking Sign out clears the token from the store', async () => {
    useAppStore.setState({ token: 'some-token', companyId: null, agentId: null })
    server.use(http.get('/api/companies', () => HttpResponse.json([])))
    renderCompanyList()
    await screen.findByRole('button', { name: /sign out/i })
    fireEvent.click(screen.getByRole('button', { name: /sign out/i }))
    expect(useAppStore.getState().token).toBeNull()
  })
})
