import { describe, it, expect, beforeAll, afterEach, afterAll } from 'vitest'
import { render, screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { http, HttpResponse } from 'msw'
import { server } from '@/test/server'
import { AgentChat } from './index'

const COMPANY_ID = 'company-1'
const AGENT_ID = 'agent-1'

beforeAll(() => server.listen())
afterEach(() => server.resetHandlers())
afterAll(() => server.close())

function renderChat() {
  const qc = new QueryClient({
    defaultOptions: { queries: { retry: false }, mutations: { retry: false } },
  })
  return render(
    <QueryClientProvider client={qc}>
      <AgentChat agentId={AGENT_ID} companyId={COMPANY_ID} />
    </QueryClientProvider>
  )
}

describe('AgentChat', () => {
  it('renders chat history on load', async () => {
    server.use(
      http.get(`/api/companies/${COMPANY_ID}/agents/${AGENT_ID}/chat/history`, () =>
        HttpResponse.json([
          { role: 'user', content: 'Hello agent', timestamp: '2024-01-01T00:00:00Z' },
          { role: 'assistant', content: 'Hello user', timestamp: '2024-01-01T00:00:01Z' },
        ])
      )
    )

    renderChat()

    await waitFor(() => {
      expect(screen.getByText('Hello agent')).toBeInTheDocument()
      expect(screen.getByText('Hello user')).toBeInTheDocument()
    })
  })

  it('sends a message and shows the reply', async () => {
    server.use(
      http.get(`/api/companies/${COMPANY_ID}/agents/${AGENT_ID}/chat/history`, () =>
        HttpResponse.json([])
      ),
      http.post(`/api/companies/${COMPANY_ID}/agents/${AGENT_ID}/chat`, () =>
        HttpResponse.json({ reply: 'Got your message!' })
      )
    )

    renderChat()

    const input = await screen.findByPlaceholderText('Message agent…')
    await userEvent.type(input, 'test message')
    await userEvent.click(screen.getByRole('button', { name: /send message/i }))

    await waitFor(() => {
      expect(screen.getByText('test message')).toBeInTheDocument()
    })

    await waitFor(() => {
      expect(screen.getByText('Got your message!')).toBeInTheDocument()
    })
  })

  it('pause button calls pause endpoint', async () => {
    let pauseCalled = false
    server.use(
      http.get(`/api/companies/${COMPANY_ID}/agents/${AGENT_ID}/chat/history`, () =>
        HttpResponse.json([])
      ),
      http.post(`/api/companies/${COMPANY_ID}/agents/${AGENT_ID}/pause`, () => {
        pauseCalled = true
        return HttpResponse.json({ status: 'paused' })
      })
    )

    renderChat()

    await userEvent.click(await screen.findByRole('button', { name: /pause agent/i }))

    await waitFor(() => {
      expect(pauseCalled).toBe(true)
    })
  })

  it('resume button calls resume endpoint', async () => {
    let resumeCalled = false
    server.use(
      http.get(`/api/companies/${COMPANY_ID}/agents/${AGENT_ID}/chat/history`, () =>
        HttpResponse.json([])
      ),
      http.post(`/api/companies/${COMPANY_ID}/agents/${AGENT_ID}/resume`, () => {
        resumeCalled = true
        return HttpResponse.json({ status: 'resumed' })
      })
    )

    renderChat()

    await userEvent.click(await screen.findByRole('button', { name: /resume agent/i }))

    await waitFor(() => {
      expect(resumeCalled).toBe(true)
    })
  })
})
