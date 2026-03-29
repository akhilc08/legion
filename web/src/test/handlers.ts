import { http, HttpResponse } from 'msw'

export const handlers = [
  http.post('/api/auth/login', () =>
    HttpResponse.json({ token: 'mock-token', user: { id: 'user-1', email: 'test@test.com' } })
  ),
  http.get('/api/companies', () => HttpResponse.json([])),
]
