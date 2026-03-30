import { describe, it, expect, vi, beforeEach } from 'vitest'
import { render, screen } from '@testing-library/react'

// Mock useAutoLogin – it has async side effects we don't want in App tests.
vi.mock('@/hooks/useAutoLogin', () => ({ useAutoLogin: vi.fn() }))

// Control token via this mutable variable.
let mockToken: string | null = null

vi.mock('@/store/useAppStore', () => ({
  useAppStore: (sel: (s: { token: string | null }) => unknown) =>
    sel({ token: mockToken }),
}))

// Mock page components to keep rendering shallow.
vi.mock('@/pages/CompanyList', () => ({
  CompanyList: () => <div data-testid="company-list">CompanyList</div>,
}))

vi.mock('@/pages/CompanyShell', () => ({
  CompanyShell: () => <div data-testid="company-shell">CompanyShell</div>,
}))

// Import AFTER mocks are registered.
import { App } from './App'

beforeEach(() => {
  mockToken = null
})

describe('App – unauthenticated state (no token)', () => {
  it('renders "Starting…" when token is null', () => {
    mockToken = null
    render(<App />)
    expect(screen.getByText('Starting…')).toBeInTheDocument()
  })

  it('does NOT render CompanyList when token is null', () => {
    mockToken = null
    render(<App />)
    expect(screen.queryByTestId('company-list')).not.toBeInTheDocument()
  })

  it('does NOT render CompanyShell when token is null', () => {
    mockToken = null
    render(<App />)
    expect(screen.queryByTestId('company-shell')).not.toBeInTheDocument()
  })
})

describe('App – authenticated state (token present)', () => {
  beforeEach(() => {
    mockToken = 'valid-token'
  })

  it('renders CompanyList at the root path', () => {
    render(<App />)
    // BrowserRouter defaults to '/' in jsdom; CompanyList should be visible.
    expect(screen.getByTestId('company-list')).toBeInTheDocument()
  })

  it('does NOT render "Starting…" when token is set', () => {
    render(<App />)
    expect(screen.queryByText('Starting…')).not.toBeInTheDocument()
  })
})

describe('App – structure', () => {
  it('renders without crashing when token is null', () => {
    mockToken = null
    expect(() => render(<App />)).not.toThrow()
  })

  it('renders without crashing when token is set', () => {
    mockToken = 'tok'
    expect(() => render(<App />)).not.toThrow()
  })
})
