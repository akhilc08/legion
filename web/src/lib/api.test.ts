import { describe, it, expect, beforeAll, afterEach, afterAll } from 'vitest'
import { server } from '@/test/server'
import { http, HttpResponse } from 'msw'
import { apiClient } from '@/lib/api'

beforeAll(() => server.listen())
afterEach(() => server.resetHandlers())
afterAll(() => server.close())

describe('apiClient', () => {
  it('attaches Authorization header when token is in localStorage', async () => {
    localStorage.setItem('legion_token', 'test-jwt')
    let capturedAuth = ''
    server.use(
      http.get('/api/companies', ({ request }) => {
        capturedAuth = request.headers.get('Authorization') ?? ''
        return HttpResponse.json([])
      })
    )
    await apiClient.get('/api/companies')
    expect(capturedAuth).toBe('Bearer test-jwt')
    localStorage.removeItem('legion_token')
  })

  it('does not attach Authorization header when no token', async () => {
    localStorage.removeItem('legion_token')
    let capturedAuth: string | null = null
    server.use(
      http.get('/api/companies', ({ request }) => {
        capturedAuth = request.headers.get('Authorization')
        return HttpResponse.json([])
      })
    )
    await apiClient.get('/api/companies')
    expect(capturedAuth).toBeNull()
  })
})
