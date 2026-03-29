import { useState } from 'react'
import { useParams } from 'react-router-dom'
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { Folder, File, ChevronRight } from 'lucide-react'
import { apiClient } from '@/lib/api'
import type { FSEntry, FSPermission, Agent } from '@/lib/types'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Badge } from '@/components/ui/badge'

const LEVEL_COLORS: Record<string, string> = {
  read: 'bg-blue-600',
  write: 'bg-yellow-600',
  admin: 'bg-red-600',
}

export function FSBrowser() {
  const { companyId } = useParams<{ companyId: string }>()
  const qc = useQueryClient()

  const [currentPath, setCurrentPath] = useState('/')
  const [grantPath, setGrantPath] = useState('/')
  const [grantAgentId, setGrantAgentId] = useState('')
  const [grantLevel, setGrantLevel] = useState<'read' | 'write' | 'admin'>('read')

  const { data: entries = [] } = useQuery<FSEntry[]>({
    queryKey: ['fs-browse', companyId, currentPath],
    queryFn: () =>
      apiClient
        .get(`/api/companies/${companyId}/fs/browse`, { params: { path: currentPath } })
        .then((r) => r.data),
    enabled: !!companyId,
  })

  const { data: permissions = [] } = useQuery<FSPermission[]>({
    queryKey: ['fs-permissions', companyId],
    queryFn: () =>
      apiClient.get(`/api/companies/${companyId}/fs/permissions`).then((r) => r.data),
    enabled: !!companyId,
  })

  const { data: agents = [] } = useQuery<Agent[]>({
    queryKey: ['agents', companyId],
    queryFn: () =>
      apiClient.get(`/api/companies/${companyId}/agents`).then((r) => r.data),
    enabled: !!companyId,
  })

  const revoke = useMutation({
    mutationFn: (permId: string) =>
      apiClient.delete(`/api/companies/${companyId}/fs/permissions/${permId}`),
    onSuccess: () => qc.invalidateQueries({ queryKey: ['fs-permissions', companyId] }),
  })

  const grant = useMutation({
    mutationFn: () =>
      apiClient.post(`/api/companies/${companyId}/fs/permissions`, {
        agent_id: grantAgentId,
        path: grantPath,
        permission_level: grantLevel,
      }),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ['fs-permissions', companyId] })
      setGrantAgentId('')
      setGrantPath(currentPath)
      setGrantLevel('read')
    },
  })

  function navigateTo(entry: FSEntry) {
    if (!entry.is_dir) return
    const next =
      currentPath === '/'
        ? `/${entry.name}`
        : `${currentPath}/${entry.name}`
    setCurrentPath(next)
    setGrantPath(next)
  }

  function navigateBreadcrumb(index: number) {
    const parts = currentPath.split('/').filter(Boolean)
    const next = index < 0 ? '/' : '/' + parts.slice(0, index + 1).join('/')
    setCurrentPath(next)
    setGrantPath(next)
  }

  const breadcrumbs = currentPath.split('/').filter(Boolean)

  function agentLabel(id: string) {
    const a = agents.find((ag) => ag.id === id)
    return a ? `${a.title} (${a.role})` : id.slice(0, 8) + '…'
  }

  function formatSize(bytes: number) {
    if (bytes === 0) return '—'
    if (bytes < 1024) return `${bytes} B`
    if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(1)} KB`
    return `${(bytes / (1024 * 1024)).toFixed(1)} MB`
  }

  return (
    <div className="flex h-full p-6 gap-6">
      {/* Left panel: File Tree */}
      <div className="w-2/5 flex flex-col gap-3">
        <h1 className="text-lg font-semibold text-zinc-100">File System</h1>

        {/* Breadcrumb */}
        <div className="flex items-center gap-1 text-xs text-zinc-400 flex-wrap">
          <button
            className="hover:text-zinc-200 transition-colors"
            onClick={() => navigateBreadcrumb(-1)}
          >
            /
          </button>
          {breadcrumbs.map((part, i) => (
            <span key={i} className="flex items-center gap-1">
              <ChevronRight className="h-3 w-3 text-zinc-600" />
              <button
                className="hover:text-zinc-200 transition-colors"
                onClick={() => navigateBreadcrumb(i)}
              >
                {part}
              </button>
            </span>
          ))}
        </div>

        <div className="rounded-md border border-zinc-800 overflow-hidden flex-1">
          <div className="bg-zinc-900 px-3 py-2 border-b border-zinc-800">
            <span className="text-xs text-zinc-500 font-mono">{currentPath}</span>
          </div>
          <div className="divide-y divide-zinc-800">
            {entries.length === 0 && (
              <div className="px-4 py-6 text-center text-sm text-zinc-600">Empty directory</div>
            )}
            {entries.map((entry) => (
              <div
                key={entry.name}
                className={`flex items-center gap-3 px-3 py-2 text-sm transition-colors ${
                  entry.is_dir
                    ? 'cursor-pointer hover:bg-zinc-800/60'
                    : 'cursor-default'
                }`}
                onClick={() => navigateTo(entry)}
              >
                {entry.is_dir ? (
                  <Folder className="h-4 w-4 text-yellow-500 shrink-0" />
                ) : (
                  <File className="h-4 w-4 text-zinc-500 shrink-0" />
                )}
                <span className="flex-1 text-zinc-300 truncate">{entry.name}</span>
                {!entry.is_dir && (
                  <span className="text-xs text-zinc-600">{formatSize(entry.size)}</span>
                )}
              </div>
            ))}
          </div>
        </div>
      </div>

      {/* Right panel: Permissions */}
      <div className="w-3/5 flex flex-col gap-4">
        <h1 className="text-lg font-semibold text-zinc-100">Access Control</h1>

        <div className="rounded-md border border-zinc-800 overflow-hidden">
          <table className="w-full">
            <thead>
              <tr className="border-b border-zinc-800 bg-zinc-900">
                <th className="px-3 py-2 text-left text-xs font-medium text-zinc-500">Agent</th>
                <th className="px-3 py-2 text-left text-xs font-medium text-zinc-500">Path</th>
                <th className="px-3 py-2 text-left text-xs font-medium text-zinc-500">Level</th>
                <th className="px-3 py-2 text-left text-xs font-medium text-zinc-500">Actions</th>
              </tr>
            </thead>
            <tbody>
              {permissions.map((perm) => (
                <tr key={perm.id} className="border-b border-zinc-800 hover:bg-zinc-900/50">
                  <td className="px-3 py-2 text-xs text-zinc-300">{agentLabel(perm.agent_id)}</td>
                  <td className="px-3 py-2 text-xs font-mono text-zinc-400 truncate max-w-[120px]">
                    {perm.path}
                  </td>
                  <td className="px-3 py-2">
                    <Badge className={`${LEVEL_COLORS[perm.permission_level]} text-white text-xs border-0`}>
                      {perm.permission_level}
                    </Badge>
                  </td>
                  <td className="px-3 py-2">
                    <Button
                      size="sm"
                      variant="destructive"
                      className="h-6 px-2 text-xs"
                      onClick={() => revoke.mutate(perm.id)}
                      disabled={revoke.isPending}
                    >
                      Revoke
                    </Button>
                  </td>
                </tr>
              ))}
              {permissions.length === 0 && (
                <tr>
                  <td colSpan={4} className="px-3 py-6 text-center text-sm text-zinc-600">
                    No permissions granted.
                  </td>
                </tr>
              )}
            </tbody>
          </table>
        </div>

        {/* Grant Permission Form */}
        <div className="rounded-md border border-zinc-800 bg-zinc-900/40 p-4">
          <h2 className="text-sm font-medium text-zinc-300 mb-3">Grant Permission</h2>
          <div className="flex flex-col gap-3">
            <div className="flex gap-2">
              <div className="flex-1">
                <label className="text-xs text-zinc-500 mb-1 block">Path</label>
                <Input
                  value={grantPath}
                  onChange={(e) => setGrantPath(e.target.value)}
                  className="h-8 text-xs bg-zinc-900 border-zinc-700"
                  placeholder="/"
                />
              </div>
              <div className="flex-1">
                <label className="text-xs text-zinc-500 mb-1 block">Level</label>
                <select
                  value={grantLevel}
                  onChange={(e) => setGrantLevel(e.target.value as 'read' | 'write' | 'admin')}
                  className="h-8 w-full rounded-md border border-zinc-700 bg-zinc-900 px-2 text-xs text-zinc-300 focus:outline-none focus:ring-1 focus:ring-zinc-600"
                >
                  <option value="read">read</option>
                  <option value="write">write</option>
                  <option value="admin">admin</option>
                </select>
              </div>
            </div>
            <div>
              <label className="text-xs text-zinc-500 mb-1 block">Agent</label>
              <select
                value={grantAgentId}
                onChange={(e) => setGrantAgentId(e.target.value)}
                className="h-8 w-full rounded-md border border-zinc-700 bg-zinc-900 px-2 text-xs text-zinc-300 focus:outline-none focus:ring-1 focus:ring-zinc-600"
              >
                <option value="">Select agent…</option>
                {agents.map((a) => (
                  <option key={a.id} value={a.id}>
                    {a.title} ({a.role})
                  </option>
                ))}
              </select>
            </div>
            <Button
              size="sm"
              className="self-end"
              disabled={!grantAgentId || !grantPath || grant.isPending}
              onClick={() => grant.mutate()}
            >
              Grant
            </Button>
          </div>
        </div>
      </div>
    </div>
  )
}
