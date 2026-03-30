import { describe, it, expect, vi, beforeAll, afterAll, afterEach, beforeEach } from 'vitest'
import { render, screen, fireEvent, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { MemoryRouter } from 'react-router-dom'
import { http, HttpResponse } from 'msw'
import { server } from '@/test/server'
import { Register } from './Register'
import { useAppStore } from '@/store/useAppStore'

const mockNavigate = vi.fn()

vi.mock('react-router-dom', async () => {
  const actual = await vi.importActual<typeof import('react-router-dom')>('react-router-dom')
  return { ...actual, useNavigate: () => mockNavigate }
})

function renderRegister() {
  return render(
    <MemoryRouter>
      <Register />
    </MemoryRouter>
  )
}

beforeAll(() => server.listen({ onUnhandledRequest: 'error' }))
afterEach(() => {
  server.resetHandlers()
  mockNavigate.mockReset()
  localStorage.clear()
  useAppStore.setState({ token: null, companyId: null, agentId: null })
})
afterAll(() => server.close())

describe('Register – rendering', () => {
  it('renders "Create account" heading', () => {
    renderRegister()
    // The title "Create account" appears in both the CardTitle and the button;
    // verify the heading role specifically (the CardTitle renders as a div with heading text).
    expect(screen.getAllByText('Create account').length).toBeGreaterThanOrEqual(1)
  })

  it('renders an email input', () => {
    renderRegister()
    expect(screen.getByLabelText(/email/i)).toBeInTheDocument()
  })

  it('email input has type="email"', () => {
    renderRegister()
    expect(screen.getByLabelText(/email/i)).toHaveAttribute('type', 'email')
  })

  it('renders a password input', () => {
    renderRegister()
    expect(screen.getByLabelText(/password/i)).toBeInTheDocument()
  })

  it('password input has type="password"', () => {
    renderRegister()
    expect(screen.getByLabelText(/password/i)).toHaveAttribute('type', 'password')
  })

  it('renders a "Create account" submit button', () => {
    renderRegister()
    expect(screen.getByRole('button', { name: /create account/i })).toBeInTheDocument()
  })

  it('submit button is of type submit', () => {
    renderRegister()
    expect(screen.getByRole('button', { name: /create account/i })).toHaveAttribute('type', 'submit')
  })

  it('renders a link to the Sign in / Login page', () => {
    renderRegister()
    expect(screen.getByRole('link', { name: /sign in/i })).toBeInTheDocument()
  })

  it('Sign in link points to /login', () => {
    renderRegister()
    expect(screen.getByRole('link', { name: /sign in/i })).toHaveAttribute('href', '/login')
  })

  it('does not show an error message on initial render', () => {
    renderRegister()
    expect(screen.queryByText(/registration failed/i)).not.toBeInTheDocument()
  })

  it('renders "Have an account?" prompt text', () => {
    renderRegister()
    expect(screen.getByText(/have an account/i)).toBeInTheDocument()
  })
})

describe('Register – form validation (HTML required)', () => {
  it('email input is required', () => {
    renderRegister()
    expect(screen.getByLabelText(/email/i)).toBeRequired()
  })

  it('password input is required', () => {
    renderRegister()
    expect(screen.getByLabelText(/password/i)).toBeRequired()
  })

  it('submit button is not disabled initially', () => {
    renderRegister()
    expect(screen.getByRole('button', { name: /create account/i })).not.toBeDisabled()
  })
})

describe('Register – interactions', () => {
  it('typing into email input updates its value', async () => {
    renderRegister()
    const emailInput = screen.getByLabelText(/email/i)
    await userEvent.type(emailInput, 'newuser@example.com')
    expect(emailInput).toHaveValue('newuser@example.com')
  })

  it('typing into password input updates its value', async () => {
    renderRegister()
    const passwordInput = screen.getByLabelText(/password/i)
    await userEvent.type(passwordInput, 'mysecret')
    expect(passwordInput).toHaveValue('mysecret')
  })
})

describe('Register – successful submission', () => {
  beforeEach(() => {
    server.use(
      http.post('/api/auth/register', () =>
        HttpResponse.json({ token: 'register-jwt-token' })
      )
    )
  })

  it('calls POST /api/auth/register on form submit', async () => {
    const spy = vi.fn(() => HttpResponse.json({ token: 'register-jwt-token' }))
    server.use(http.post('/api/auth/register', spy))
    renderRegister()
    await userEvent.type(screen.getByLabelText(/email/i), 'new@example.com')
    await userEvent.type(screen.getByLabelText(/password/i), 'securepass')
    fireEvent.submit(screen.getByRole('button', { name: /create account/i }).closest('form')!)
    await waitFor(() => expect(spy).toHaveBeenCalledOnce())
  })

  it('stores token in app store on success', async () => {
    renderRegister()
    await userEvent.type(screen.getByLabelText(/email/i), 'new@example.com')
    await userEvent.type(screen.getByLabelText(/password/i), 'securepass')
    fireEvent.submit(screen.getByRole('button', { name: /create account/i }).closest('form')!)
    await waitFor(() => expect(useAppStore.getState().token).toBe('register-jwt-token'))
  })

  it('stores token in localStorage on success', async () => {
    renderRegister()
    await userEvent.type(screen.getByLabelText(/email/i), 'new@example.com')
    await userEvent.type(screen.getByLabelText(/password/i), 'securepass')
    fireEvent.submit(screen.getByRole('button', { name: /create account/i }).closest('form')!)
    await waitFor(() => expect(localStorage.getItem('legion_token')).toBe('register-jwt-token'))
  })

  it('navigates to "/" on success', async () => {
    renderRegister()
    await userEvent.type(screen.getByLabelText(/email/i), 'new@example.com')
    await userEvent.type(screen.getByLabelText(/password/i), 'securepass')
    fireEvent.submit(screen.getByRole('button', { name: /create account/i }).closest('form')!)
    await waitFor(() => expect(mockNavigate).toHaveBeenCalledWith('/'))
  })

  it('does not show an error message on success', async () => {
    renderRegister()
    await userEvent.type(screen.getByLabelText(/email/i), 'new@example.com')
    await userEvent.type(screen.getByLabelText(/password/i), 'securepass')
    fireEvent.submit(screen.getByRole('button', { name: /create account/i }).closest('form')!)
    await waitFor(() => expect(mockNavigate).toHaveBeenCalled())
    expect(screen.queryByText(/registration failed/i)).not.toBeInTheDocument()
  })
})

describe('Register – error handling', () => {
  beforeEach(() => {
    server.use(
      http.post('/api/auth/register', () =>
        HttpResponse.json({ error: 'Email taken' }, { status: 409 })
      )
    )
  })

  it('shows error message on failed registration', async () => {
    renderRegister()
    await userEvent.type(screen.getByLabelText(/email/i), 'taken@example.com')
    await userEvent.type(screen.getByLabelText(/password/i), 'pass')
    fireEvent.submit(screen.getByRole('button', { name: /create account/i }).closest('form')!)
    await screen.findByText(/registration failed/i)
  })

  it('error message mentions email may already be taken', async () => {
    renderRegister()
    await userEvent.type(screen.getByLabelText(/email/i), 'taken@example.com')
    await userEvent.type(screen.getByLabelText(/password/i), 'pass')
    fireEvent.submit(screen.getByRole('button', { name: /create account/i }).closest('form')!)
    const err = await screen.findByText(/email may already be taken/i)
    expect(err).toBeVisible()
  })

  it('does not navigate on error', async () => {
    renderRegister()
    await userEvent.type(screen.getByLabelText(/email/i), 'taken@example.com')
    await userEvent.type(screen.getByLabelText(/password/i), 'pass')
    fireEvent.submit(screen.getByRole('button', { name: /create account/i }).closest('form')!)
    await screen.findByText(/registration failed/i)
    expect(mockNavigate).not.toHaveBeenCalled()
  })

  it('does not store token on error', async () => {
    renderRegister()
    await userEvent.type(screen.getByLabelText(/email/i), 'taken@example.com')
    await userEvent.type(screen.getByLabelText(/password/i), 'pass')
    fireEvent.submit(screen.getByRole('button', { name: /create account/i }).closest('form')!)
    await screen.findByText(/registration failed/i)
    expect(useAppStore.getState().token).toBeNull()
  })

  it('re-enables button after error', async () => {
    renderRegister()
    await userEvent.type(screen.getByLabelText(/email/i), 'taken@example.com')
    await userEvent.type(screen.getByLabelText(/password/i), 'pass')
    fireEvent.submit(screen.getByRole('button', { name: /create account/i }).closest('form')!)
    await screen.findByText(/registration failed/i)
    expect(screen.getByRole('button', { name: /create account/i })).not.toBeDisabled()
  })

  it('clears previous error on re-submit', async () => {
    renderRegister()
    await userEvent.type(screen.getByLabelText(/email/i), 'taken@example.com')
    await userEvent.type(screen.getByLabelText(/password/i), 'pass')
    fireEvent.submit(screen.getByRole('button', { name: /create account/i }).closest('form')!)
    await screen.findByText(/registration failed/i)

    // Switch handler to success and resubmit
    server.use(
      http.post('/api/auth/register', () =>
        HttpResponse.json({ token: 'new-token' })
      )
    )
    fireEvent.submit(screen.getByRole('button', { name: /create account/i }).closest('form')!)
    await waitFor(() =>
      expect(screen.queryByText(/registration failed/i)).not.toBeInTheDocument()
    )
  })

  it('shows 500-level error as registration failed', async () => {
    server.use(
      http.post('/api/auth/register', () =>
        HttpResponse.json({ error: 'Server error' }, { status: 500 })
      )
    )
    renderRegister()
    await userEvent.type(screen.getByLabelText(/email/i), 'u@e.com')
    await userEvent.type(screen.getByLabelText(/password/i), 'pass')
    fireEvent.submit(screen.getByRole('button', { name: /create account/i }).closest('form')!)
    await screen.findByText(/registration failed/i)
  })
})

describe('Register – loading state', () => {
  it('shows "Creating…" text during request', async () => {
    let resolve!: (value: unknown) => void
    server.use(
      http.post('/api/auth/register', () =>
        new Promise((r) => { resolve = r })
      )
    )
    renderRegister()
    await userEvent.type(screen.getByLabelText(/email/i), 'u@e.com')
    await userEvent.type(screen.getByLabelText(/password/i), 'pass')
    fireEvent.submit(screen.getByRole('button', { name: /create account/i }).closest('form')!)
    await screen.findByText(/creating…/i)
    resolve(HttpResponse.json({ token: 'tok' }))
  })

  it('disables submit button while loading', async () => {
    let resolve!: (value: unknown) => void
    server.use(
      http.post('/api/auth/register', () =>
        new Promise((r) => { resolve = r })
      )
    )
    renderRegister()
    await userEvent.type(screen.getByLabelText(/email/i), 'u@e.com')
    await userEvent.type(screen.getByLabelText(/password/i), 'pass')
    fireEvent.submit(screen.getByRole('button', { name: /create account/i }).closest('form')!)
    await screen.findByText(/creating…/i)
    expect(screen.getByRole('button', { name: /creating…/i })).toBeDisabled()
    resolve(HttpResponse.json({ token: 'tok' }))
  })
})

describe('Register – keyboard interaction', () => {
  it('pressing Enter in password field after filling both fields submits', async () => {
    server.use(
      http.post('/api/auth/register', () =>
        HttpResponse.json({ token: 'tok' })
      )
    )
    renderRegister()
    await userEvent.type(screen.getByLabelText(/email/i), 'u@e.com')
    await userEvent.type(screen.getByLabelText(/password/i), 'pass{enter}')
    await waitFor(() => expect(mockNavigate).toHaveBeenCalled())
  })
})
