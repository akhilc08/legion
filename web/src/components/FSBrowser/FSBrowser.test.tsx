import { describe, it, expect, beforeAll, afterEach, afterAll, vi } from 'vitest'
import { render, screen, waitFor, within } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { MemoryRouter, Route, Routes } from 'react-router-dom'
import { server } from '@/test/server'
import { http, HttpResponse } from 'msw'
import { FSBrowser } from './index'
import type { Agent, FSEntry, FSPermission } from '@/lib/types'

beforeAll(() => server.listen())
afterEach(() => server.resetHandlers())
afterAll(() => server.close())

function makeWrapper(companyId = 'c1') {
  const qc = new QueryClient({ defaultOptions: { queries: { retry: false }, mutations: { retry: false } } })
  return ({ children }: { children: React.ReactNode }) => (
    <QueryClientProvider client={qc}>
      <MemoryRouter initialEntries={[`/companies/${companyId}/files`]}>
        <Routes><Route path="/companies/:companyId/files" element={<>{children}</>} /></Routes>
      </MemoryRouter>
    </QueryClientProvider>
  )
}

const makeAgent = (id: string, title: string, role: string): Agent => ({
  id, company_id: 'c1', role, title, system_prompt: '', manager_id: null,
  runtime: 'claude_code', status: 'idle', monthly_budget: 10000, token_spend: 0,
  chat_token_spend: 0, pid: null,
  created_at: new Date().toISOString(), updated_at: new Date().toISOString(),
})

const makePermission = (id: string, agentId: string, path: string, level: FSPermission['permission_level']): FSPermission => ({
  id, agent_id: agentId, path, permission_level: level, granted_by: null,
})

function setupDefaults(
  entries: FSEntry[] = [],
  permissions: FSPermission[] = [],
  agents: Agent[] = [],
) {
  server.use(
    http.get('/api/companies/:id/fs/browse', () => HttpResponse.json(entries)),
    http.get('/api/companies/:id/fs/permissions', () => HttpResponse.json(permissions)),
    http.get('/api/companies/:id/agents', () => HttpResponse.json(agents)),
  )
}

// ── Headings ──────────────────────────────────────────────────────────────────

describe('FSBrowser headings', () => {
  it('renders "File System" heading', async () => {
    setupDefaults()
    render(<FSBrowser />, { wrapper: makeWrapper() })
    expect(await screen.findByText('File System')).toBeInTheDocument()
  })

  it('renders "Access Control" heading', async () => {
    setupDefaults()
    render(<FSBrowser />, { wrapper: makeWrapper() })
    expect(await screen.findByText('Access Control')).toBeInTheDocument()
  })
})

// ── Directory entries ─────────────────────────────────────────────────────────

describe('FSBrowser file/directory entries', () => {
  it('empty directory shows "Empty directory" message', async () => {
    setupDefaults([])
    render(<FSBrowser />, { wrapper: makeWrapper() })
    expect(await screen.findByText('Empty directory')).toBeInTheDocument()
  })

  it('file entries listed with names', async () => {
    setupDefaults([
      { name: 'README.md', is_dir: false, size: 1024 },
      { name: 'config.json', is_dir: false, size: 512 },
    ])
    render(<FSBrowser />, { wrapper: makeWrapper() })
    expect(await screen.findByText('README.md')).toBeInTheDocument()
    expect(await screen.findByText('config.json')).toBeInTheDocument()
  })

  it('directory entries listed', async () => {
    setupDefaults([
      { name: 'reports', is_dir: true, size: 0 },
      { name: 'logs', is_dir: true, size: 0 },
    ])
    render(<FSBrowser />, { wrapper: makeWrapper() })
    expect(await screen.findByText('reports')).toBeInTheDocument()
    expect(await screen.findByText('logs')).toBeInTheDocument()
  })

  it('file size 0 formatted as "—"', async () => {
    setupDefaults([{ name: 'empty.txt', is_dir: false, size: 0 }])
    render(<FSBrowser />, { wrapper: makeWrapper() })
    await screen.findByText('empty.txt')
    expect(screen.getByText('—')).toBeInTheDocument()
  })

  it('file size 512 formatted as "512 B"', async () => {
    setupDefaults([{ name: 'small.txt', is_dir: false, size: 512 }])
    render(<FSBrowser />, { wrapper: makeWrapper() })
    await screen.findByText('small.txt')
    expect(screen.getByText('512 B')).toBeInTheDocument()
  })

  it('file size 1500 formatted as "1.5 KB"', async () => {
    setupDefaults([{ name: 'medium.txt', is_dir: false, size: 1500 }])
    render(<FSBrowser />, { wrapper: makeWrapper() })
    await screen.findByText('medium.txt')
    expect(screen.getByText('1.5 KB')).toBeInTheDocument()
  })

  it('file size 2MB formatted as "2.0 MB"', async () => {
    setupDefaults([{ name: 'large.bin', is_dir: false, size: 2 * 1024 * 1024 }])
    render(<FSBrowser />, { wrapper: makeWrapper() })
    await screen.findByText('large.bin')
    expect(screen.getByText('2.0 MB')).toBeInTheDocument()
  })

  it('file size is not shown for directory entries', async () => {
    setupDefaults([{ name: 'mydir', is_dir: true, size: 0 }])
    render(<FSBrowser />, { wrapper: makeWrapper() })
    await screen.findByText('mydir')
    // directories do not show a size span — no "—" or "0 B"
    expect(screen.queryByText('—')).not.toBeInTheDocument()
  })
})

// ── Navigation ────────────────────────────────────────────────────────────────

describe('FSBrowser navigation', () => {
  it('clicking a directory entry navigates into it (updates currentPath)', async () => {
    // First render shows root with one dir
    server.use(
      http.get('/api/companies/:id/fs/browse', ({ request }) => {
        const url = new URL(request.url)
        const path = url.searchParams.get('path') ?? '/'
        if (path === '/') return HttpResponse.json([{ name: 'docs', is_dir: true, size: 0 }])
        return HttpResponse.json([{ name: 'file.txt', is_dir: false, size: 100 }])
      }),
      http.get('/api/companies/:id/fs/permissions', () => HttpResponse.json([])),
      http.get('/api/companies/:id/agents', () => HttpResponse.json([])),
    )
    render(<FSBrowser />, { wrapper: makeWrapper() })
    await userEvent.click(await screen.findByText('docs'))
    expect(await screen.findByText('file.txt')).toBeInTheDocument()
  })

  it('clicking a file entry does nothing (no navigation away from current entries)', async () => {
    setupDefaults([
      { name: 'README.md', is_dir: false, size: 1024 },
    ])
    render(<FSBrowser />, { wrapper: makeWrapper() })
    await screen.findByText('README.md')
    await userEvent.click(screen.getByText('README.md'))
    // Should still be on the same view
    expect(screen.getByText('README.md')).toBeInTheDocument()
  })

  it('breadcrumb shows "/" at root', async () => {
    setupDefaults()
    render(<FSBrowser />, { wrapper: makeWrapper() })
    await screen.findByText('File System')
    // The root breadcrumb button is "/"
    expect(screen.getByRole('button', { name: '/' })).toBeInTheDocument()
  })

  it('clicking "/" in breadcrumb returns to root', async () => {
    server.use(
      http.get('/api/companies/:id/fs/browse', ({ request }) => {
        const url = new URL(request.url)
        const path = url.searchParams.get('path') ?? '/'
        if (path === '/') return HttpResponse.json([{ name: 'src', is_dir: true, size: 0 }])
        return HttpResponse.json([{ name: 'nested.txt', is_dir: false, size: 50 }])
      }),
      http.get('/api/companies/:id/fs/permissions', () => HttpResponse.json([])),
      http.get('/api/companies/:id/agents', () => HttpResponse.json([])),
    )
    render(<FSBrowser />, { wrapper: makeWrapper() })
    await userEvent.click(await screen.findByText('src'))
    await screen.findByText('nested.txt')
    await userEvent.click(screen.getByRole('button', { name: '/' }))
    expect(await screen.findByText('src')).toBeInTheDocument()
  })

  it('clicking a breadcrumb segment navigates to that level', async () => {
    server.use(
      http.get('/api/companies/:id/fs/browse', ({ request }) => {
        const url = new URL(request.url)
        const path = url.searchParams.get('path') ?? '/'
        if (path === '/') return HttpResponse.json([{ name: 'a', is_dir: true, size: 0 }])
        if (path === '/a') return HttpResponse.json([{ name: 'b', is_dir: true, size: 0 }])
        return HttpResponse.json([{ name: 'leaf.txt', is_dir: false, size: 10 }])
      }),
      http.get('/api/companies/:id/fs/permissions', () => HttpResponse.json([])),
      http.get('/api/companies/:id/agents', () => HttpResponse.json([])),
    )
    render(<FSBrowser />, { wrapper: makeWrapper() })
    await userEvent.click(await screen.findByText('a'))
    await userEvent.click(await screen.findByText('b'))
    await screen.findByText('leaf.txt')
    // Click breadcrumb "a" to go back to /a
    await userEvent.click(screen.getByRole('button', { name: 'a' }))
    expect(await screen.findByText('b')).toBeInTheDocument()
  })
})

// ── Permissions table ─────────────────────────────────────────────────────────

describe('FSBrowser permissions', () => {
  it('"No permissions granted." when permissions empty', async () => {
    setupDefaults([], [])
    render(<FSBrowser />, { wrapper: makeWrapper() })
    expect(await screen.findByText('No permissions granted.')).toBeInTheDocument()
  })

  it('permissions table: agent name shown', async () => {
    const agent = makeAgent('ag1', 'Data Analyst', 'analyst')
    setupDefaults([], [makePermission('p1', 'ag1', '/data', 'read')], [agent])
    render(<FSBrowser />, { wrapper: makeWrapper() })
    expect((await screen.findAllByText('Data Analyst (analyst)')).length).toBeGreaterThanOrEqual(1)
  })

  it('permissions table: path shown', async () => {
    const agent = makeAgent('ag1', 'Data Analyst', 'analyst')
    setupDefaults([], [makePermission('p1', 'ag1', '/data/reports', 'read')], [agent])
    render(<FSBrowser />, { wrapper: makeWrapper() })
    expect(await screen.findByText('/data/reports')).toBeInTheDocument()
  })

  it('permissions table: permission level shown', async () => {
    const agent = makeAgent('ag1', 'Data Analyst', 'analyst')
    setupDefaults([], [makePermission('p1', 'ag1', '/data', 'write')], [agent])
    render(<FSBrowser />, { wrapper: makeWrapper() })
    expect(await screen.findByText('write')).toBeInTheDocument()
  })

  it('Revoke button present', async () => {
    const agent = makeAgent('ag1', 'Alice', 'engineer')
    setupDefaults([], [makePermission('p1', 'ag1', '/src', 'read')], [agent])
    render(<FSBrowser />, { wrapper: makeWrapper() })
    expect(await screen.findByRole('button', { name: 'Revoke' })).toBeInTheDocument()
  })

  it('Revoke button calls DELETE /permissions/:id', async () => {
    let revokedId: string | null = null
    const agent = makeAgent('ag1', 'Alice', 'engineer')
    setupDefaults([], [makePermission('p-special', 'ag1', '/src', 'read')], [agent])
    server.use(
      http.delete('/api/companies/:companyId/fs/permissions/:permId', ({ params }) => {
        revokedId = params.permId as string
        return HttpResponse.json({ ok: true })
      }),
    )
    render(<FSBrowser />, { wrapper: makeWrapper() })
    await userEvent.click(await screen.findByRole('button', { name: 'Revoke' }))
    await waitFor(() => expect(revokedId).toBe('p-special'))
  })

  it('on revoke success: refetches permissions', async () => {
    let permCallCount = 0
    const agent = makeAgent('ag1', 'Alice', 'engineer')
    server.use(
      http.get('/api/companies/:id/fs/browse', () => HttpResponse.json([])),
      http.get('/api/companies/:id/fs/permissions', () => {
        permCallCount++
        return HttpResponse.json([makePermission('p1', 'ag1', '/src', 'read')])
      }),
      http.get('/api/companies/:id/agents', () => HttpResponse.json([agent])),
      http.delete('/api/companies/:id/fs/permissions/:permId', () =>
        HttpResponse.json({ ok: true }),
      ),
    )
    render(<FSBrowser />, { wrapper: makeWrapper() })
    await userEvent.click(await screen.findByRole('button', { name: 'Revoke' }))
    await waitFor(() => expect(permCallCount).toBeGreaterThan(1))
  })

  it('permission level badge: read shown', async () => {
    const agent = makeAgent('ag1', 'Alice', 'engineer')
    setupDefaults([], [makePermission('p1', 'ag1', '/src', 'read')], [agent])
    render(<FSBrowser />, { wrapper: makeWrapper() })
    expect(await screen.findByText('read')).toBeInTheDocument()
  })

  it('permission level badge: write shown', async () => {
    const agent = makeAgent('ag1', 'Alice', 'engineer')
    setupDefaults([], [makePermission('p1', 'ag1', '/src', 'write')], [agent])
    render(<FSBrowser />, { wrapper: makeWrapper() })
    expect(await screen.findByText('write')).toBeInTheDocument()
  })

  it('permission level badge: admin shown', async () => {
    const agent = makeAgent('ag1', 'Alice', 'engineer')
    setupDefaults([], [makePermission('p1', 'ag1', '/src', 'admin')], [agent])
    render(<FSBrowser />, { wrapper: makeWrapper() })
    expect(await screen.findByText('admin')).toBeInTheDocument()
  })
})

// ── Grant form ────────────────────────────────────────────────────────────────

describe('FSBrowser grant form', () => {
  it('Path input present', async () => {
    setupDefaults()
    render(<FSBrowser />, { wrapper: makeWrapper() })
    await screen.findByText('Grant Permission')
    // The path input has placeholder "/"
    expect(screen.getByPlaceholderText('/')).toBeInTheDocument()
  })

  it('Level select present with read/write/admin options', async () => {
    setupDefaults()
    render(<FSBrowser />, { wrapper: makeWrapper() })
    await screen.findByText('Grant Permission')
    expect(screen.getByRole('option', { name: 'read' })).toBeInTheDocument()
    expect(screen.getByRole('option', { name: 'write' })).toBeInTheDocument()
    expect(screen.getByRole('option', { name: 'admin' })).toBeInTheDocument()
  })

  it('Agent select present with placeholder', async () => {
    setupDefaults()
    render(<FSBrowser />, { wrapper: makeWrapper() })
    await screen.findByText('Grant Permission')
    expect(screen.getByRole('option', { name: 'Select agent…' })).toBeInTheDocument()
  })

  it('Grant button disabled when no agent selected', async () => {
    setupDefaults()
    render(<FSBrowser />, { wrapper: makeWrapper() })
    await screen.findByText('Grant Permission')
    expect(screen.getByRole('button', { name: 'Grant' })).toBeDisabled()
  })

  it('Grant button disabled when path empty', async () => {
    const agent = makeAgent('ag1', 'Alice', 'engineer')
    setupDefaults([], [], [agent])
    render(<FSBrowser />, { wrapper: makeWrapper() })
    await screen.findByText('Grant Permission')
    // Clear the path input
    const pathInput = screen.getByPlaceholderText('/')
    await userEvent.clear(pathInput)
    // Select an agent
    const agentSelect = screen.getAllByRole('combobox').find(
      s => within(s).queryByText('Select agent…') !== null
    )
    await userEvent.selectOptions(agentSelect!, 'ag1')
    expect(screen.getByRole('button', { name: 'Grant' })).toBeDisabled()
  })

  it('Grant button enabled when both path and agent filled', async () => {
    const agent = makeAgent('ag1', 'Alice', 'engineer')
    setupDefaults([], [], [agent])
    render(<FSBrowser />, { wrapper: makeWrapper() })
    await screen.findByText('Grant Permission')
    await screen.findByRole('option', { name: 'Alice (engineer)' })
    const agentSelect = screen.getAllByRole('combobox').find(
      s => within(s).queryByText('Select agent…') !== null
    )
    await userEvent.selectOptions(agentSelect!, 'ag1')
    expect(screen.getByRole('button', { name: 'Grant' })).not.toBeDisabled()
  })

  it('clicking Grant calls POST /permissions', async () => {
    let grantHit = false
    const agent = makeAgent('ag1', 'Alice', 'engineer')
    server.use(
      http.get('/api/companies/:id/fs/browse', () => HttpResponse.json([])),
      http.get('/api/companies/:id/fs/permissions', () => HttpResponse.json([])),
      http.get('/api/companies/:id/agents', () => HttpResponse.json([agent])),
      http.post('/api/companies/:id/fs/permissions', () => {
        grantHit = true
        return HttpResponse.json({ id: 'new-perm' })
      }),
    )
    render(<FSBrowser />, { wrapper: makeWrapper() })
    await screen.findByText('Grant Permission')
    await screen.findByRole('option', { name: 'Alice (engineer)' })
    const agentSelect = screen.getAllByRole('combobox').find(
      s => within(s).queryByText('Select agent…') !== null
    )
    await userEvent.selectOptions(agentSelect!, 'ag1')
    await userEvent.click(screen.getByRole('button', { name: 'Grant' }))
    await waitFor(() => expect(grantHit).toBe(true))
  })

  it('on grant success: form resets (agent select cleared)', async () => {
    const agent = makeAgent('ag1', 'Alice', 'engineer')
    server.use(
      http.get('/api/companies/:id/fs/browse', () => HttpResponse.json([])),
      http.get('/api/companies/:id/fs/permissions', () => HttpResponse.json([])),
      http.get('/api/companies/:id/agents', () => HttpResponse.json([agent])),
      http.post('/api/companies/:id/fs/permissions', () =>
        HttpResponse.json({ id: 'new-perm' }),
      ),
    )
    render(<FSBrowser />, { wrapper: makeWrapper() })
    await screen.findByText('Grant Permission')
    await screen.findByRole('option', { name: 'Alice (engineer)' })
    const agentSelect = screen.getAllByRole('combobox').find(
      s => within(s).queryByText('Select agent…') !== null
    ) as HTMLSelectElement
    await userEvent.selectOptions(agentSelect, 'ag1')
    await userEvent.click(screen.getByRole('button', { name: 'Grant' }))
    await waitFor(() => expect(agentSelect.value).toBe(''))
  })

  it('on grant success: refetches permissions', async () => {
    let permCallCount = 0
    const agent = makeAgent('ag1', 'Alice', 'engineer')
    server.use(
      http.get('/api/companies/:id/fs/browse', () => HttpResponse.json([])),
      http.get('/api/companies/:id/fs/permissions', () => {
        permCallCount++
        return HttpResponse.json([])
      }),
      http.get('/api/companies/:id/agents', () => HttpResponse.json([agent])),
      http.post('/api/companies/:id/fs/permissions', () =>
        HttpResponse.json({ id: 'new-perm' }),
      ),
    )
    render(<FSBrowser />, { wrapper: makeWrapper() })
    await screen.findByText('Grant Permission')
    await screen.findByRole('option', { name: 'Alice (engineer)' })
    const agentSelect = screen.getAllByRole('combobox').find(
      s => within(s).queryByText('Select agent…') !== null
    )
    await userEvent.selectOptions(agentSelect!, 'ag1')
    await userEvent.click(screen.getByRole('button', { name: 'Grant' }))
    await waitFor(() => expect(permCallCount).toBeGreaterThan(1))
  })

  it('Level select options: read', async () => {
    setupDefaults()
    render(<FSBrowser />, { wrapper: makeWrapper() })
    await screen.findByText('Grant Permission')
    expect(screen.getByRole('option', { name: 'read' })).toBeInTheDocument()
  })

  it('Level select options: write', async () => {
    setupDefaults()
    render(<FSBrowser />, { wrapper: makeWrapper() })
    await screen.findByText('Grant Permission')
    expect(screen.getByRole('option', { name: 'write' })).toBeInTheDocument()
  })

  it('Level select options: admin', async () => {
    setupDefaults()
    render(<FSBrowser />, { wrapper: makeWrapper() })
    await screen.findByText('Grant Permission')
    expect(screen.getByRole('option', { name: 'admin' })).toBeInTheDocument()
  })
})

// ── agentLabel ────────────────────────────────────────────────────────────────

describe('agentLabel in permissions table', () => {
  it('shows "Title (role)" when agent found', async () => {
    const agent = makeAgent('ag1', 'Data Analyst', 'analyst')
    setupDefaults([], [makePermission('p1', 'ag1', '/data', 'read')], [agent])
    render(<FSBrowser />, { wrapper: makeWrapper() })
    expect((await screen.findAllByText('Data Analyst (analyst)')).length).toBeGreaterThanOrEqual(1)
  })

  it('shows truncated UUID when agent not found', async () => {
    const longId = 'abcdef12-1234-1234-1234-abcdefabcdef'
    setupDefaults([], [makePermission('p1', longId, '/data', 'read')], [])
    render(<FSBrowser />, { wrapper: makeWrapper() })
    // agentLabel truncates to first 8 chars + '…'
    expect(await screen.findByText('abcdef12…')).toBeInTheDocument()
  })
})
