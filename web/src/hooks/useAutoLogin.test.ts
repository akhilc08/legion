import { describe, it, expect, beforeAll, afterEach, afterAll, beforeEach, vi } from 'vitest'
import { renderHook, waitFor, act } from '@testing-library/react'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { MemoryRouter } from 'react-router-dom'
import { createElement, type ReactNode } from 'react'
import { http, HttpResponse } from 'msw'
import { server } from '@/test/server'
import { useAutoLogin } from './useAutoLogin'
import { useAppStore } from '@/store/useAppStore'

beforeAll(() => server.listen({ onUnhandledRequest: 'bypass' }))
afterEach(() => {
  server.resetHandlers()
  localStorage.clear()
  act(() => {
    useAppStore.setState({ token: null, companyId: null, agentId: null })
  })
})
afterAll(() => server.close())

function makeWrapper() {
  const qc = new QueryClient({ defaultOptions: { queries: { retry: false } } })
  return function Wrapper({ children }: { children: ReactNode }) {
    return createElement(
      MemoryRouter,
      null,
      createElement(QueryClientProvider, { client: qc }, children)
    )
  }
}

// Helper: block the default login handler so no test bleeds mock-token
function blockDefaultLogin() {
  server.use(http.post('/api/auth/login', () => new HttpResponse(null, { status: 403 })))
}

describe('useAutoLogin – token already present', () => {
  beforeEach(() => {
    act(() => {
      useAppStore.setState({ token: 'existing-tok', companyId: null, agentId: null })
    })
    localStorage.setItem('legion_token', 'existing-tok')
    blockDefaultLogin()
  })

  it('does NOT call /api/auth/login when token is in store', async () => {
    let loginCalled = false
    server.use(
      http.post('/api/auth/login', () => {
        loginCalled = true
        return HttpResponse.json({ token: 'new-tok' })
      })
    )
    const { unmount } = renderHook(() => useAutoLogin(), { wrapper: makeWrapper() })
    await new Promise((r) => setTimeout(r, 80))
    unmount()
    expect(loginCalled).toBe(false)
  })

  it('does NOT call /api/auth/register when token is in store', async () => {
    let registerCalled = false
    server.use(
      http.post('/api/auth/register', () => {
        registerCalled = true
        return HttpResponse.json({})
      })
    )
    const { unmount } = renderHook(() => useAutoLogin(), { wrapper: makeWrapper() })
    await new Promise((r) => setTimeout(r, 80))
    unmount()
    expect(registerCalled).toBe(false)
  })

  it('token remains unchanged when already present', async () => {
    const { unmount } = renderHook(() => useAutoLogin(), { wrapper: makeWrapper() })
    await new Promise((r) => setTimeout(r, 80))
    unmount()
    expect(useAppStore.getState().token).toBe('existing-tok')
  })
})

describe('useAutoLogin – no token, login succeeds', () => {
  it('calls POST /api/auth/login when no token', async () => {
    let loginCalled = false
    server.use(
      http.post('/api/auth/login', () => {
        loginCalled = true
        return HttpResponse.json({ token: 'fresh-tok' })
      })
    )
    const { unmount } = renderHook(() => useAutoLogin(), { wrapper: makeWrapper() })
    await waitFor(() => expect(loginCalled).toBe(true), { timeout: 2000 })
    unmount()
  })

  it('uses default email admin@legion.local', async () => {
    let capturedBody: Record<string, string> | null = null
    server.use(
      http.post('/api/auth/login', async ({ request }) => {
        capturedBody = await request.json() as Record<string, string>
        return HttpResponse.json({ token: 'tok' })
      })
    )
    const { unmount } = renderHook(() => useAutoLogin(), { wrapper: makeWrapper() })
    await waitFor(() => expect(capturedBody).not.toBeNull(), { timeout: 2000 })
    unmount()
    expect(capturedBody?.email).toBe('admin@legion.local')
  })

  it('uses default password "legion"', async () => {
    let capturedBody: Record<string, string> | null = null
    server.use(
      http.post('/api/auth/login', async ({ request }) => {
        capturedBody = await request.json() as Record<string, string>
        return HttpResponse.json({ token: 'tok' })
      })
    )
    const { unmount } = renderHook(() => useAutoLogin(), { wrapper: makeWrapper() })
    await waitFor(() => expect(capturedBody).not.toBeNull(), { timeout: 2000 })
    unmount()
    expect(capturedBody?.password).toBe('legion')
  })

  it('sets token in store on successful login', async () => {
    server.use(
      http.post('/api/auth/login', () => HttpResponse.json({ token: 'stored-tok' }))
    )
    const { unmount } = renderHook(() => useAutoLogin(), { wrapper: makeWrapper() })
    await waitFor(() => expect(useAppStore.getState().token).toBe('stored-tok'), { timeout: 2000 })
    unmount()
  })

  it('persists token to localStorage on successful login', async () => {
    server.use(
      http.post('/api/auth/login', () => HttpResponse.json({ token: 'ls-tok' }))
    )
    const { unmount } = renderHook(() => useAutoLogin(), { wrapper: makeWrapper() })
    await waitFor(() => expect(localStorage.getItem('legion_token')).toBe('ls-tok'), { timeout: 2000 })
    unmount()
  })
})

describe('useAutoLogin – login fails, register + re-login flow', () => {
  // Always block the default login handler; each test supplies its own
  beforeEach(() => { blockDefaultLogin() })

  it('calls /api/auth/register when login returns 404', async () => {
    let registerCalled = false
    server.use(
      http.post('/api/auth/login', () => new HttpResponse(null, { status: 404 })),
      http.post('/api/auth/register', () => {
        registerCalled = true
        return new HttpResponse(null, { status: 500 }) // prevent second login from succeeding
      })
    )
    const { unmount } = renderHook(() => useAutoLogin(), { wrapper: makeWrapper() })
    await waitFor(() => expect(registerCalled).toBe(true), { timeout: 2000 })
    unmount()
  })

  it('calls /api/auth/register when login returns 401', async () => {
    let registerCalled = false
    server.use(
      http.post('/api/auth/login', () => new HttpResponse(null, { status: 401 })),
      http.post('/api/auth/register', () => {
        registerCalled = true
        return new HttpResponse(null, { status: 500 })
      })
    )
    const { unmount } = renderHook(() => useAutoLogin(), { wrapper: makeWrapper() })
    await waitFor(() => expect(registerCalled).toBe(true), { timeout: 2000 })
    unmount()
  })

  it('calls login again after successful register', async () => {
    let loginCount = 0
    server.use(
      http.post('/api/auth/login', () => {
        loginCount++
        if (loginCount === 1) return new HttpResponse(null, { status: 404 })
        return HttpResponse.json({ token: 'post-register-tok' })
      }),
      http.post('/api/auth/register', () => HttpResponse.json({}))
    )
    const { unmount } = renderHook(() => useAutoLogin(), { wrapper: makeWrapper() })
    await waitFor(() => expect(loginCount).toBeGreaterThanOrEqual(2), { timeout: 2000 })
    unmount()
  })

  it('sets token after register + second login succeeds', async () => {
    let loginCount = 0
    server.use(
      http.post('/api/auth/login', () => {
        loginCount++
        if (loginCount === 1) return new HttpResponse(null, { status: 404 })
        return HttpResponse.json({ token: 'post-register-tok' })
      }),
      http.post('/api/auth/register', () => HttpResponse.json({}))
    )
    const { unmount } = renderHook(() => useAutoLogin(), { wrapper: makeWrapper() })
    await waitFor(() => expect(useAppStore.getState().token).toBe('post-register-tok'), { timeout: 2000 })
    unmount()
  })

  it('does not throw when register succeeds but second login fails', async () => {
    server.use(
      http.post('/api/auth/login', () => new HttpResponse(null, { status: 500 })),
      http.post('/api/auth/register', () => HttpResponse.json({}))
    )
    const { unmount } = renderHook(() => useAutoLogin(), { wrapper: makeWrapper() })
    await new Promise((r) => setTimeout(r, 300))
    unmount()
    await new Promise((r) => setTimeout(r, 30))
    expect(useAppStore.getState().token).toBeNull()
  })

  it('does not throw when both register and second login fail', async () => {
    server.use(
      http.post('/api/auth/login', () => new HttpResponse(null, { status: 500 })),
      http.post('/api/auth/register', () => new HttpResponse(null, { status: 500 }))
    )
    const { unmount } = renderHook(() => useAutoLogin(), { wrapper: makeWrapper() })
    await new Promise((r) => setTimeout(r, 200))
    unmount()
    await new Promise((r) => setTimeout(r, 30))
    expect(useAppStore.getState().token).toBeNull()
  })

  it('all errors caught: no unhandled promise rejection', async () => {
    server.use(
      http.post('/api/auth/login', () => HttpResponse.error()),
      http.post('/api/auth/register', () => HttpResponse.error())
    )
    const unhandled = vi.fn()
    process.once('unhandledRejection', unhandled)
    const { unmount } = renderHook(() => useAutoLogin(), { wrapper: makeWrapper() })
    await new Promise((r) => setTimeout(r, 150))
    unmount()
    process.removeListener('unhandledRejection', unhandled)
    expect(unhandled).not.toHaveBeenCalled()
  })
})
