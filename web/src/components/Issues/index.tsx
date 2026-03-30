import { useState } from 'react'
import { useParams } from 'react-router-dom'
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { apiClient } from '@/lib/api'
import type { Agent, Issue, IssueStatus } from '@/lib/types'
import { IssueRow } from './IssueRow'
import { Input } from '@/components/ui/input'
import { Button } from '@/components/ui/button'

const ALL_STATUSES: IssueStatus[] = ['pending', 'in_progress', 'blocked', 'done', 'failed']

export function Issues() {
  const { companyId } = useParams<{ companyId: string }>()
  const qc = useQueryClient()
  const [search, setSearch] = useState('')
  const [statusFilter, setStatusFilter] = useState<IssueStatus | ''>('')
  const [showForm, setShowForm] = useState(false)
  const [form, setForm] = useState({ title: '', description: '', assignee_id: '', parent_id: '' })

  const { data: issues = [] } = useQuery<Issue[]>({
    queryKey: ['issues', companyId],
    queryFn: () => apiClient.get(`/api/companies/${companyId}/issues`).then((r) => r.data),
    enabled: !!companyId,
  })
  const { data: rawAgents = [] } = useQuery<Agent[]>({
    queryKey: ['agents', companyId],
    queryFn: () => apiClient.get(`/api/companies/${companyId}/agents`).then((r) => r.data),
    enabled: !!companyId,
  })
  const agents = [...rawAgents].sort((a, b) =>
    a.role === 'board' ? -1 : b.role === 'board' ? 1 : 0
  )

  const create = useMutation({
    mutationFn: () =>
      apiClient.post(`/api/companies/${companyId}/issues`, {
        title: form.title,
        description: form.description,
        assignee_id: form.assignee_id || null,
        parent_id: form.parent_id || null,
      }),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ['issues', companyId] })
      setForm({ title: '', description: '', assignee_id: '', parent_id: '' })
      setShowForm(false)
    },
  })

  const filtered = (Array.isArray(issues) ? issues : []).filter((i) => {
    const matchSearch = i.title.toLowerCase().includes(search.toLowerCase())
    const matchStatus = statusFilter ? i.status === statusFilter : true
    return matchSearch && matchStatus
  })

  return (
    <div className="p-6 max-w-5xl">
      <div className="flex items-center justify-between mb-4">
        <h1 className="text-lg font-semibold text-zinc-100">Issues</h1>
        <Button size="sm" onClick={() => setShowForm((v) => !v)} variant={showForm ? 'secondary' : 'default'}>
          {showForm ? 'Cancel' : 'New Issue'}
        </Button>
      </div>

      {showForm && (
        <div className="mb-4 rounded-md border border-zinc-700 bg-zinc-900 p-4 space-y-3">
          <Input
            placeholder="Title"
            value={form.title}
            onChange={(e) => setForm((f) => ({ ...f, title: e.target.value }))}
            className="bg-zinc-800 border-zinc-700 text-zinc-100"
          />
          <textarea
            placeholder="Description (optional)"
            value={form.description}
            onChange={(e) => setForm((f) => ({ ...f, description: e.target.value }))}
            rows={3}
            className="w-full rounded-md border border-zinc-700 bg-zinc-800 text-zinc-100 text-sm px-3 py-2 resize-none placeholder:text-zinc-500 focus:outline-none focus:ring-1 focus:ring-zinc-500"
          />
          <div className="flex gap-3">
            <select
              value={form.assignee_id}
              onChange={(e) => setForm((f) => ({ ...f, assignee_id: e.target.value }))}
              className="flex-1 rounded-md border border-zinc-700 bg-zinc-800 text-zinc-300 text-sm px-3 py-2"
            >
              <option value="">No assignee</option>
              {agents.map((a) => (
                <option key={a.id} value={a.id}>{a.title}</option>
              ))}
            </select>
            <select
              value={form.parent_id}
              onChange={(e) => setForm((f) => ({ ...f, parent_id: e.target.value }))}
              className="flex-1 rounded-md border border-zinc-700 bg-zinc-800 text-zinc-300 text-sm px-3 py-2"
            >
              <option value="">No parent issue</option>
              {issues.map((i) => (
                <option key={i.id} value={i.id}>{i.title}</option>
              ))}
            </select>
          </div>
          <div className="flex justify-end">
            <Button
              size="sm"
              disabled={!form.title.trim() || create.isPending}
              onClick={() => create.mutate()}
            >
              {create.isPending ? 'Creating…' : 'Create Issue'}
            </Button>
          </div>
          {create.isError && (
            <p className="text-xs text-red-400">{String((create.error as Error).message)}</p>
          )}
        </div>
      )}

      <div className="flex gap-3 mb-4">
        <Input placeholder="Search issues…" value={search} onChange={(e) => setSearch(e.target.value)} className="max-w-xs bg-zinc-800 border-zinc-700 text-zinc-100" />
        <select value={statusFilter} onChange={(e) => setStatusFilter(e.target.value as IssueStatus | '')} className="rounded-md border border-zinc-700 bg-zinc-800 text-zinc-300 text-sm px-3 py-2">
          <option value="">All statuses</option>
          {ALL_STATUSES.map((s) => <option key={s} value={s}>{s}</option>)}
        </select>
      </div>
      <div className="rounded-md border border-zinc-800 overflow-hidden">
        <table className="w-full">
          <thead>
            <tr className="border-b border-zinc-800 bg-zinc-900">
              <th className="px-3 py-2 text-left text-xs font-medium text-zinc-500">Title</th>
              <th className="px-3 py-2 text-left text-xs font-medium text-zinc-500">Status</th>
              <th className="px-3 py-2 text-left text-xs font-medium text-zinc-500">Assignee</th>
              <th className="px-3 py-2 text-left text-xs font-medium text-zinc-500">Attempts</th>
              <th className="w-6" />
            </tr>
          </thead>
          <tbody>
            {filtered.map((issue) => <IssueRow key={issue.id} issue={issue} agents={agents} />)}
            {filtered.length === 0 && <tr><td colSpan={5} className="px-3 py-6 text-center text-sm text-zinc-600">No issues found.</td></tr>}
          </tbody>
        </table>
      </div>
    </div>
  )
}
