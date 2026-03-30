import { describe, it, expect, beforeAll, afterEach, afterAll, beforeEach, vi } from 'vitest'
import { server } from '@/test/server'
import { http, HttpResponse } from 'msw'
import { apiClient } from '@/lib/api'

beforeAll(() => server.listen())
afterEach(() => {
  server.resetHandlers()
  localStorage.clear()
  vi.unstubAllGlobals()
})
afterAll(() => server.close())

// Helper: make a unique GET endpoint to avoid handler conflicts
function useEndpoint(path: string, handler: Parameters<typeof http.get>[1]) {
  server.use(http.get(path, handler))
}

describe('apiClient – Authorization header', () => {
  it('attaches Bearer token when token is in localStorage', async () => {
    localStorage.setItem('legion_token', 'tok-abc')
    let captured = ''
    useEndpoint('/api/companies', ({ request }) => {
      captured = request.headers.get('Authorization') ?? ''
      return HttpResponse.json([])
    })
    await apiClient.get('/api/companies')
    expect(captured).toBe('Bearer tok-abc')
  })

  it('does NOT attach Authorization header when no token', async () => {
    let captured: string | null = 'present'
    useEndpoint('/api/companies', ({ request }) => {
      captured = request.headers.get('Authorization')
      return HttpResponse.json([])
    })
    await apiClient.get('/api/companies')
    expect(captured).toBeNull()
  })

  it('picks up a token set after client was created (interceptor reads live)', async () => {
    let captured = ''
    useEndpoint('/api/companies', ({ request }) => {
      captured = request.headers.get('Authorization') ?? ''
      return HttpResponse.json([])
    })
    // First request: no token
    await apiClient.get('/api/companies')
    expect(captured).toBe('')

    // Set token then next request carries it
    localStorage.setItem('legion_token', 'late-token')
    await apiClient.get('/api/companies')
    expect(captured).toBe('Bearer late-token')
  })

  it('multiple concurrent requests all get the auth header', async () => {
    localStorage.setItem('legion_token', 'concurrent-tok')
    const captured: string[] = []
    server.use(
      http.get('/api/r1', ({ request }) => { captured.push(request.headers.get('Authorization') ?? ''); return HttpResponse.json({}) }),
      http.get('/api/r2', ({ request }) => { captured.push(request.headers.get('Authorization') ?? ''); return HttpResponse.json({}) }),
      http.get('/api/r3', ({ request }) => { captured.push(request.headers.get('Authorization') ?? ''); return HttpResponse.json({}) }),
    )
    await Promise.all([apiClient.get('/api/r1'), apiClient.get('/api/r2'), apiClient.get('/api/r3')])
    expect(captured).toHaveLength(3)
    captured.forEach((h) => expect(h).toBe('Bearer concurrent-tok'))
  })
})

describe('apiClient – baseURL and Content-Type', () => {
  it('baseURL is "/"', () => {
    expect((apiClient.defaults.baseURL)).toBe('/')
  })

  it('Content-Type header is application/json by default', () => {
    const ct = apiClient.defaults.headers['Content-Type'] as string
    expect(ct).toBe('application/json')
  })
})

describe('apiClient – 200 / 500 / network error', () => {
  it('resolves normally on 200 response', async () => {
    server.use(http.get('/api/ok', () => HttpResponse.json({ ok: true })))
    const res = await apiClient.get('/api/ok')
    expect(res.status).toBe(200)
    expect(res.data).toEqual({ ok: true })
  })

  it('rejects on 500 response', async () => {
    server.use(http.get('/api/err', () => new HttpResponse(null, { status: 500 })))
    await expect(apiClient.get('/api/err')).rejects.toThrow()
  })

  it('rejects on network error', async () => {
    server.use(http.get('/api/net', () => HttpResponse.error()))
    await expect(apiClient.get('/api/net')).rejects.toThrow()
  })

  it('rejects on 404 response', async () => {
    server.use(http.get('/api/missing', () => new HttpResponse(null, { status: 404 })))
    await expect(apiClient.get('/api/missing')).rejects.toThrow()
  })

  it('rejects on 400 response', async () => {
    server.use(http.post('/api/bad', () => new HttpResponse(null, { status: 400 })))
    await expect(apiClient.post('/api/bad', {})).rejects.toThrow()
  })
})

describe('apiClient – 401 redirect logic', () => {
  beforeEach(() => {
    localStorage.setItem('legion_token', 'valid-tok')
  })

  it('on 401 from non-auth endpoint: removes token and redirects to "/"', async () => {
    const loc = { href: 'http://localhost/app' }
    Object.defineProperty(window, 'location', { value: loc, writable: true, configurable: true })
    server.use(http.get('/api/protected', () => new HttpResponse(null, { status: 401 })))
    try {
      await apiClient.get('/api/protected')
    } catch {
      // expected rejection
    }
    expect(localStorage.getItem('legion_token')).toBeNull()
    expect(loc.href).toBe('/')
  })

  it('on 401 from non-auth endpoint: sets window.location.href to "/"', async () => {
    const loc = { href: 'http://localhost/dashboard' }
    Object.defineProperty(window, 'location', { value: loc, writable: true, configurable: true })
    server.use(http.get('/api/guarded', () => new HttpResponse(null, { status: 401 })))
    try { await apiClient.get('/api/guarded') } catch { /* expected */ }
    expect(loc.href).toBe('/')
  })

  it('on 401 from /api/auth/login: does NOT remove token', async () => {
    server.use(http.post('/api/auth/login', () => new HttpResponse(null, { status: 401 })))
    try { await apiClient.post('/api/auth/login', {}) } catch { /* expected */ }
    expect(localStorage.getItem('legion_token')).toBe('valid-tok')
  })

  it('on 401 from /api/auth/login: does NOT redirect', async () => {
    const loc = { href: 'http://localhost/app' }
    Object.defineProperty(window, 'location', { value: loc, writable: true, configurable: true })
    server.use(http.post('/api/auth/login', () => new HttpResponse(null, { status: 401 })))
    try { await apiClient.post('/api/auth/login', {}) } catch { /* expected */ }
    expect(loc.href).toBe('http://localhost/app')
  })

  it('on 401 from /api/auth/register: does NOT remove token', async () => {
    server.use(http.post('/api/auth/register', () => new HttpResponse(null, { status: 401 })))
    try { await apiClient.post('/api/auth/register', {}) } catch { /* expected */ }
    expect(localStorage.getItem('legion_token')).toBe('valid-tok')
  })

  it('on 401 from /api/auth/register: does NOT redirect', async () => {
    const loc = { href: 'http://localhost/app' }
    Object.defineProperty(window, 'location', { value: loc, writable: true, configurable: true })
    server.use(http.post('/api/auth/register', () => new HttpResponse(null, { status: 401 })))
    try { await apiClient.post('/api/auth/register', {}) } catch { /* expected */ }
    expect(loc.href).toBe('http://localhost/app')
  })

  it('on 401 from /api/auth/logout (any auth sub-path): does NOT redirect', async () => {
    const loc = { href: 'http://localhost/app' }
    Object.defineProperty(window, 'location', { value: loc, writable: true, configurable: true })
    server.use(http.post('/api/auth/logout', () => new HttpResponse(null, { status: 401 })))
    try { await apiClient.post('/api/auth/logout', {}) } catch { /* expected */ }
    expect(loc.href).toBe('http://localhost/app')
  })

  it('on 401 non-auth: rejects the promise', async () => {
    server.use(http.get('/api/reject-me', () => new HttpResponse(null, { status: 401 })))
    await expect(apiClient.get('/api/reject-me')).rejects.toThrow()
  })

  it('after 401 triggers token clear, a new token can be set and is picked up', async () => {
    // Use a stub that keeps href as a full URL so jsdom can still resolve requests
    const loc = { href: 'http://localhost/' }
    Object.defineProperty(window, 'location', { value: loc, writable: true, configurable: true })

    server.use(http.get('/api/trigger401b', () => new HttpResponse(null, { status: 401 })))
    try { await apiClient.get('/api/trigger401b') } catch { /* expected */ }
    expect(localStorage.getItem('legion_token')).toBeNull()

    // Restore a valid href so axios can resolve the relative baseURL
    loc.href = 'http://localhost/'
    localStorage.setItem('legion_token', 'new-token')
    let capturedAuth = ''
    server.use(http.get('/api/after401', ({ request }) => {
      capturedAuth = request.headers.get('Authorization') ?? ''
      return HttpResponse.json({ ok: true })
    }))
    const res = await apiClient.get('/api/after401')
    expect(res.status).toBe(200)
    expect(capturedAuth).toBe('Bearer new-token')
  })

  it('non-401 errors from non-auth endpoints do not clear token', async () => {
    server.use(http.get('/api/server-error', () => new HttpResponse(null, { status: 500 })))
    try { await apiClient.get('/api/server-error') } catch { /* expected */ }
    expect(localStorage.getItem('legion_token')).toBe('valid-tok')
  })

  it('non-401 errors from non-auth endpoints do not redirect', async () => {
    const loc = { href: 'http://localhost/app' }
    Object.defineProperty(window, 'location', { value: loc, writable: true, configurable: true })
    server.use(http.get('/api/server-error2', () => new HttpResponse(null, { status: 503 })))
    try { await apiClient.get('/api/server-error2') } catch { /* expected */ }
    expect(loc.href).toBe('http://localhost/app')
  })

  it('403 from non-auth endpoint does NOT redirect', async () => {
    const loc = { href: 'http://localhost/app' }
    Object.defineProperty(window, 'location', { value: loc, writable: true, configurable: true })
    server.use(http.get('/api/forbidden', () => new HttpResponse(null, { status: 403 })))
    try { await apiClient.get('/api/forbidden') } catch { /* expected */ }
    expect(loc.href).toBe('http://localhost/app')
  })
})

describe('apiClient – POST / PUT / DELETE methods', () => {
  it('POST resolves on 201', async () => {
    server.use(http.post('/api/items', () => HttpResponse.json({ id: '1' }, { status: 201 })))
    const res = await apiClient.post('/api/items', { name: 'test' })
    expect(res.status).toBe(201)
  })

  it('PUT resolves on 200', async () => {
    server.use(http.put('/api/items/1', () => HttpResponse.json({ id: '1' })))
    const res = await apiClient.put('/api/items/1', { name: 'updated' })
    expect(res.status).toBe(200)
  })

  it('DELETE resolves on 204', async () => {
    server.use(http.delete('/api/items/1', () => new HttpResponse(null, { status: 204 })))
    const res = await apiClient.delete('/api/items/1')
    expect(res.status).toBe(204)
  })

  it('PATCH resolves on 200', async () => {
    server.use(http.patch('/api/items/1', () => HttpResponse.json({ patched: true })))
    const res = await apiClient.patch('/api/items/1', { field: 'v' })
    expect(res.data).toEqual({ patched: true })
  })
})
