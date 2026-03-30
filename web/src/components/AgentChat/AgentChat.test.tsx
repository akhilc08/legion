import { describe, it, expect, beforeAll, afterEach, afterAll, vi } from 'vitest'
import { render, screen, waitFor, within } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { http, HttpResponse } from 'msw'
import { server } from '@/test/server'
import { AgentChat } from './index'

const COMPANY_ID = 'company-1'
const AGENT_ID = 'agent-1'

const HISTORY_URL = `/api/companies/${COMPANY_ID}/agents/${AGENT_ID}/chat/history`
const CHAT_URL = `/api/companies/${COMPANY_ID}/agents/${AGENT_ID}/chat`
const PAUSE_URL = `/api/companies/${COMPANY_ID}/agents/${AGENT_ID}/pause`
const RESUME_URL = `/api/companies/${COMPANY_ID}/agents/${AGENT_ID}/resume`

beforeAll(() => server.listen())
afterEach(() => server.resetHandlers())
afterAll(() => server.close())

function makeQC() {
  return new QueryClient({
    defaultOptions: { queries: { retry: false }, mutations: { retry: false } },
  })
}

function renderChat(qc?: QueryClient) {
  const client = qc ?? makeQC()
  return render(
    <QueryClientProvider client={client}>
      <AgentChat agentId={AGENT_ID} companyId={COMPANY_ID} />
    </QueryClientProvider>
  )
}

function emptyHistory() {
  return http.get(HISTORY_URL, () => HttpResponse.json([]))
}

describe('AgentChat – initial render', () => {
  it('renders "No messages yet" when history is empty', async () => {
    server.use(emptyHistory())
    renderChat()
    await waitFor(() =>
      expect(screen.getByText('No messages yet')).toBeInTheDocument()
    )
  })

  it('shows loading spinner while history is loading', () => {
    server.use(
      http.get(HISTORY_URL, () => new Promise(() => { /* never resolves */ }))
    )
    renderChat()
    // Loader2 renders as an svg; check for the animate-spin class
    const spinner = document.querySelector('.animate-spin')
    expect(spinner).toBeInTheDocument()
  })

  it('does not show "No messages yet" while loading', () => {
    server.use(
      http.get(HISTORY_URL, () => new Promise(() => { /* never resolves */ }))
    )
    renderChat()
    expect(screen.queryByText('No messages yet')).not.toBeInTheDocument()
  })

  it('renders loaded chat history – user message', async () => {
    server.use(
      http.get(HISTORY_URL, () =>
        HttpResponse.json([
          { role: 'user', content: 'Hello agent', timestamp: '2024-01-01T00:00:00Z' },
        ])
      )
    )
    renderChat()
    await waitFor(() =>
      expect(screen.getByText('Hello agent')).toBeInTheDocument()
    )
  })

  it('renders loaded chat history – assistant message', async () => {
    server.use(
      http.get(HISTORY_URL, () =>
        HttpResponse.json([
          { role: 'assistant', content: 'Hello user', timestamp: '2024-01-01T00:00:01Z' },
        ])
      )
    )
    renderChat()
    await waitFor(() =>
      expect(screen.getByText('Hello user')).toBeInTheDocument()
    )
  })

  it('renders both user and assistant messages', async () => {
    server.use(
      http.get(HISTORY_URL, () =>
        HttpResponse.json([
          { role: 'user', content: 'Hi there', timestamp: '2024-01-01T00:00:00Z' },
          { role: 'assistant', content: 'Hi back', timestamp: '2024-01-01T00:00:01Z' },
        ])
      )
    )
    renderChat()
    await waitFor(() => {
      expect(screen.getByText('Hi there')).toBeInTheDocument()
      expect(screen.getByText('Hi back')).toBeInTheDocument()
    })
  })

  it('renders multiple messages in order', async () => {
    server.use(
      http.get(HISTORY_URL, () =>
        HttpResponse.json([
          { role: 'user', content: 'Msg 1', timestamp: '2024-01-01T00:00:00Z' },
          { role: 'assistant', content: 'Msg 2', timestamp: '2024-01-01T00:00:01Z' },
          { role: 'user', content: 'Msg 3', timestamp: '2024-01-01T00:00:02Z' },
        ])
      )
    )
    renderChat()
    await waitFor(() => {
      const msgs = screen.getAllByText(/Msg \d/)
      expect(msgs).toHaveLength(3)
      expect(msgs[0]).toHaveTextContent('Msg 1')
      expect(msgs[1]).toHaveTextContent('Msg 2')
      expect(msgs[2]).toHaveTextContent('Msg 3')
    })
  })
})

describe('AgentChat – message alignment', () => {
  it('user messages appear right-aligned (justify-end)', async () => {
    server.use(
      http.get(HISTORY_URL, () =>
        HttpResponse.json([
          { role: 'user', content: 'Right side', timestamp: '2024-01-01T00:00:00Z' },
        ])
      )
    )
    const { container } = renderChat()
    await waitFor(() => expect(screen.getByText('Right side')).toBeInTheDocument())
    const msgEl = screen.getByText('Right side')
    const wrapper = msgEl.closest('[class*="flex"]')
    expect(wrapper?.className).toContain('justify-end')
  })

  it('assistant messages appear left-aligned (justify-start)', async () => {
    server.use(
      http.get(HISTORY_URL, () =>
        HttpResponse.json([
          { role: 'assistant', content: 'Left side', timestamp: '2024-01-01T00:00:00Z' },
        ])
      )
    )
    renderChat()
    await waitFor(() => expect(screen.getByText('Left side')).toBeInTheDocument())
    const msgEl = screen.getByText('Left side')
    const wrapper = msgEl.closest('[class*="flex"]')
    expect(wrapper?.className).toContain('justify-start')
  })
})

describe('AgentChat – send button state', () => {
  it('send button is disabled when input is empty', async () => {
    server.use(emptyHistory())
    renderChat()
    await screen.findByPlaceholderText('Message agent…')
    const btn = screen.getByRole('button', { name: /send message/i })
    expect(btn).toBeDisabled()
  })

  it('send button is enabled when input has text', async () => {
    server.use(emptyHistory())
    renderChat()
    const input = await screen.findByPlaceholderText('Message agent…')
    await userEvent.type(input, 'hello')
    const btn = screen.getByRole('button', { name: /send message/i })
    expect(btn).not.toBeDisabled()
  })

  it('send button disabled for whitespace-only input', async () => {
    server.use(emptyHistory())
    renderChat()
    const input = await screen.findByPlaceholderText('Message agent…')
    await userEvent.type(input, '   ')
    const btn = screen.getByRole('button', { name: /send message/i })
    expect(btn).toBeDisabled()
  })

  it('send button disabled while mutation is pending', async () => {
    server.use(
      emptyHistory(),
      http.post(CHAT_URL, () => new Promise(() => { /* never resolves */ }))
    )
    renderChat()
    const input = await screen.findByPlaceholderText('Message agent…')
    await userEvent.type(input, 'test')
    await userEvent.click(screen.getByRole('button', { name: /send message/i }))
    await waitFor(() =>
      expect(screen.getByRole('button', { name: /send message/i })).toBeDisabled()
    )
  })
})

describe('AgentChat – sending messages', () => {
  it('clicking send calls POST /chat with trimmed message', async () => {
    let capturedBody: unknown
    server.use(
      emptyHistory(),
      http.post(CHAT_URL, async ({ request }) => {
        capturedBody = await request.json()
        return HttpResponse.json({ reply: 'ok' })
      })
    )
    renderChat()
    const input = await screen.findByPlaceholderText('Message agent…')
    await userEvent.type(input, '  hello world  ')
    await userEvent.click(screen.getByRole('button', { name: /send message/i }))
    await waitFor(() => expect(capturedBody).toEqual({ message: 'hello world' }))
  })

  it('pressing Enter sends the message', async () => {
    let called = false
    server.use(
      emptyHistory(),
      http.post(CHAT_URL, () => {
        called = true
        return HttpResponse.json({ reply: 'ok' })
      })
    )
    renderChat()
    const input = await screen.findByPlaceholderText('Message agent…')
    await userEvent.type(input, 'enter test{Enter}')
    await waitFor(() => expect(called).toBe(true))
  })

  it('pressing Shift+Enter does NOT send', async () => {
    let called = false
    server.use(
      emptyHistory(),
      http.post(CHAT_URL, () => {
        called = true
        return HttpResponse.json({ reply: 'ok' })
      })
    )
    renderChat()
    const input = await screen.findByPlaceholderText('Message agent…')
    await userEvent.type(input, 'no send{Shift>}{Enter}{/Shift}')
    // give async ops a moment to flush
    await new Promise((r) => setTimeout(r, 50))
    expect(called).toBe(false)
  })

  it('input is cleared after send', async () => {
    server.use(
      emptyHistory(),
      http.post(CHAT_URL, () => HttpResponse.json({ reply: 'done' }))
    )
    renderChat()
    const input = await screen.findByPlaceholderText('Message agent…')
    await userEvent.type(input, 'clear me')
    await userEvent.click(screen.getByRole('button', { name: /send message/i }))
    await waitFor(() => expect(input).toHaveValue(''))
  })

  it('empty input (whitespace only) does not send', async () => {
    let called = false
    server.use(
      emptyHistory(),
      http.post(CHAT_URL, () => {
        called = true
        return HttpResponse.json({ reply: 'ok' })
      })
    )
    renderChat()
    const input = await screen.findByPlaceholderText('Message agent…')
    await userEvent.type(input, '   {Enter}')
    await new Promise((r) => setTimeout(r, 50))
    expect(called).toBe(false)
  })

  it('input is disabled while mutation is pending', async () => {
    server.use(
      emptyHistory(),
      http.post(CHAT_URL, () => new Promise(() => { /* never resolves */ }))
    )
    renderChat()
    const input = await screen.findByPlaceholderText('Message agent…')
    await userEvent.type(input, 'pending test')
    await userEvent.click(screen.getByRole('button', { name: /send message/i }))
    await waitFor(() => expect(input).toBeDisabled())
  })
})

describe('AgentChat – optimistic UI', () => {
  it('user message appears immediately before response arrives', async () => {
    server.use(
      emptyHistory(),
      http.post(CHAT_URL, () => new Promise(() => { /* never resolves */ }))
    )
    renderChat()
    const input = await screen.findByPlaceholderText('Message agent…')
    await userEvent.type(input, 'optimistic message')
    await userEvent.click(screen.getByRole('button', { name: /send message/i }))
    // Message should appear optimistically without waiting for server
    expect(screen.getByText('optimistic message')).toBeInTheDocument()
  })

  it('after successful send assistant reply appears', async () => {
    server.use(
      emptyHistory(),
      http.post(CHAT_URL, () => HttpResponse.json({ reply: 'assistant reply here' }))
    )
    renderChat()
    const input = await screen.findByPlaceholderText('Message agent…')
    await userEvent.type(input, 'my question')
    await userEvent.click(screen.getByRole('button', { name: /send message/i }))
    await waitFor(() =>
      expect(screen.getByText('assistant reply here')).toBeInTheDocument()
    )
  })

  it('on send error optimistic user message is removed', async () => {
    server.use(
      emptyHistory(),
      http.post(CHAT_URL, () => HttpResponse.json({ error: 'fail' }, { status: 500 }))
    )
    renderChat()
    const input = await screen.findByPlaceholderText('Message agent…')
    await userEvent.type(input, 'error case')
    await userEvent.click(screen.getByRole('button', { name: /send message/i }))
    await waitFor(() =>
      expect(screen.queryByText('error case')).not.toBeInTheDocument()
    )
  })

  it('history and optimistic messages are combined correctly', async () => {
    server.use(
      http.get(HISTORY_URL, () =>
        HttpResponse.json([
          { role: 'user', content: 'History msg', timestamp: '2024-01-01T00:00:00Z' },
        ])
      ),
      http.post(CHAT_URL, () => new Promise(() => { /* never resolves */ }))
    )
    renderChat()
    await waitFor(() => expect(screen.getByText('History msg')).toBeInTheDocument())
    const input = screen.getByPlaceholderText('Message agent…')
    await userEvent.type(input, 'New optimistic')
    await userEvent.click(screen.getByRole('button', { name: /send message/i }))
    expect(screen.getByText('History msg')).toBeInTheDocument()
    expect(screen.getByText('New optimistic')).toBeInTheDocument()
  })

  it('loading spinner shown while mutation is pending', async () => {
    server.use(
      emptyHistory(),
      http.post(CHAT_URL, () => new Promise(() => { /* never resolves */ }))
    )
    renderChat()
    const input = await screen.findByPlaceholderText('Message agent…')
    await userEvent.type(input, 'spinner test')
    await userEvent.click(screen.getByRole('button', { name: /send message/i }))
    // The pending spinner is inside the messages area
    await waitFor(() => {
      const spinners = document.querySelectorAll('.animate-spin')
      expect(spinners.length).toBeGreaterThan(0)
    })
  })
})

describe('AgentChat – pause/resume', () => {
  it('pause button calls POST /pause endpoint', async () => {
    let pauseCalled = false
    server.use(
      emptyHistory(),
      http.post(PAUSE_URL, () => {
        pauseCalled = true
        return HttpResponse.json({ status: 'paused' })
      })
    )
    renderChat()
    await userEvent.click(await screen.findByRole('button', { name: /pause agent/i }))
    await waitFor(() => expect(pauseCalled).toBe(true))
  })

  it('resume button calls POST /resume endpoint', async () => {
    let resumeCalled = false
    server.use(
      emptyHistory(),
      http.post(RESUME_URL, () => {
        resumeCalled = true
        return HttpResponse.json({ status: 'resumed' })
      })
    )
    renderChat()
    await userEvent.click(await screen.findByRole('button', { name: /resume agent/i }))
    await waitFor(() => expect(resumeCalled).toBe(true))
  })

  it('pause button is disabled while pause mutation is pending', async () => {
    server.use(
      emptyHistory(),
      http.post(PAUSE_URL, () => new Promise(() => { /* never resolves */ }))
    )
    renderChat()
    const pauseBtn = await screen.findByRole('button', { name: /pause agent/i })
    await userEvent.click(pauseBtn)
    await waitFor(() => expect(pauseBtn).toBeDisabled())
  })

  it('resume button is not disabled initially', async () => {
    server.use(emptyHistory())
    renderChat()
    const resumeBtn = await screen.findByRole('button', { name: /resume agent/i })
    expect(resumeBtn).not.toBeDisabled()
  })

  it('pause button is not disabled initially', async () => {
    server.use(emptyHistory())
    renderChat()
    const pauseBtn = await screen.findByRole('button', { name: /pause agent/i })
    expect(pauseBtn).not.toBeDisabled()
  })

  it('header renders "Pause" text label', async () => {
    server.use(emptyHistory())
    renderChat()
    await screen.findByPlaceholderText('Message agent…')
    expect(screen.getByRole('button', { name: /pause agent/i })).toHaveTextContent('Pause')
  })

  it('header renders "Resume" text label', async () => {
    server.use(emptyHistory())
    renderChat()
    await screen.findByPlaceholderText('Message agent…')
    expect(screen.getByRole('button', { name: /resume agent/i })).toHaveTextContent('Resume')
  })
})

describe('AgentChat – structure', () => {
  it('renders the "Agent Chat" header label', async () => {
    server.use(emptyHistory())
    renderChat()
    await screen.findByPlaceholderText('Message agent…')
    expect(screen.getByText('Agent Chat')).toBeInTheDocument()
  })

  it('renders the message input placeholder', async () => {
    server.use(emptyHistory())
    renderChat()
    expect(await screen.findByPlaceholderText('Message agent…')).toBeInTheDocument()
  })

  it('renders send button', async () => {
    server.use(emptyHistory())
    renderChat()
    await screen.findByPlaceholderText('Message agent…')
    expect(screen.getByRole('button', { name: /send message/i })).toBeInTheDocument()
  })

  it('reply appears with distinct content from user message', async () => {
    server.use(
      emptyHistory(),
      http.post(CHAT_URL, () => HttpResponse.json({ reply: 'distinct reply' }))
    )
    renderChat()
    const input = await screen.findByPlaceholderText('Message agent…')
    await userEvent.type(input, 'distinct question')
    await userEvent.click(screen.getByRole('button', { name: /send message/i }))
    await waitFor(() => {
      expect(screen.getByText('distinct question')).toBeInTheDocument()
      expect(screen.getByText('distinct reply')).toBeInTheDocument()
    })
  })

  it('send does not fire again when clicking send on empty after clear', async () => {
    let callCount = 0
    server.use(
      emptyHistory(),
      http.post(CHAT_URL, () => {
        callCount++
        return HttpResponse.json({ reply: 'ok' })
      })
    )
    renderChat()
    const input = await screen.findByPlaceholderText('Message agent…')
    await userEvent.type(input, 'first')
    await userEvent.click(screen.getByRole('button', { name: /send message/i }))
    await waitFor(() => expect(input).toHaveValue(''))
    // now try clicking send again with empty input
    await userEvent.click(screen.getByRole('button', { name: /send message/i }))
    await new Promise((r) => setTimeout(r, 50))
    expect(callCount).toBe(1)
  })

  it('user message bubble has correct background class', async () => {
    server.use(
      http.get(HISTORY_URL, () =>
        HttpResponse.json([
          { role: 'user', content: 'User bubble', timestamp: '2024-01-01T00:00:00Z' },
        ])
      )
    )
    renderChat()
    await waitFor(() => expect(screen.getByText('User bubble')).toBeInTheDocument())
    const bubble = screen.getByText('User bubble').closest('div[class*="bg-zinc"]')
    expect(bubble?.className).toContain('bg-zinc-700')
  })

  it('assistant message bubble has correct background class', async () => {
    server.use(
      http.get(HISTORY_URL, () =>
        HttpResponse.json([
          { role: 'assistant', content: 'Asst bubble', timestamp: '2024-01-01T00:00:00Z' },
        ])
      )
    )
    renderChat()
    await waitFor(() => expect(screen.getByText('Asst bubble')).toBeInTheDocument())
    const bubble = screen.getByText('Asst bubble').closest('div[class*="bg-zinc"]')
    expect(bubble?.className).toContain('bg-zinc-800')
  })

  it('renders five messages without crashing', async () => {
    server.use(
      http.get(HISTORY_URL, () =>
        HttpResponse.json([
          { role: 'user', content: 'A', timestamp: '2024-01-01T00:00:00Z' },
          { role: 'assistant', content: 'B', timestamp: '2024-01-01T00:00:01Z' },
          { role: 'user', content: 'C', timestamp: '2024-01-01T00:00:02Z' },
          { role: 'assistant', content: 'D', timestamp: '2024-01-01T00:00:03Z' },
          { role: 'user', content: 'E', timestamp: '2024-01-01T00:00:04Z' },
        ])
      )
    )
    renderChat()
    await waitFor(() => {
      ;['A', 'B', 'C', 'D', 'E'].forEach((c) =>
        expect(screen.getByText(c)).toBeInTheDocument()
      )
    })
  })
})
