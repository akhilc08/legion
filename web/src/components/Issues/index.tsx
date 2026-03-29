import { useState } from 'react'
import { useParams } from 'react-router-dom'
import { useQuery } from '@tanstack/react-query'
import { apiClient } from '@/lib/api'
import type { Agent, Issue, IssueStatus } from '@/lib/types'
import { IssueRow } from './IssueRow'
import { Input } from '@/components/ui/input'

const ALL_STATUSES: IssueStatus[] = ['pending', 'in_progress', 'blocked', 'done', 'failed']

export function Issues() {
  const { companyId } = useParams<{ companyId: string }>()
  const [search, setSearch] = useState('')
  const [statusFilter, setStatusFilter] = useState<IssueStatus | ''>('')

  const { data: issues = [] } = useQuery<Issue[]>({
    queryKey: ['issues', companyId],
    queryFn: () => apiClient.get(`/api/companies/${companyId}/issues`).then((r) => r.data),
    enabled: !!companyId,
  })
  const { data: agents = [] } = useQuery<Agent[]>({
    queryKey: ['agents', companyId],
    queryFn: () => apiClient.get(`/api/companies/${companyId}/agents`).then((r) => r.data),
    enabled: !!companyId,
  })

  const filtered = issues.filter((i) => {
    const matchSearch = i.title.toLowerCase().includes(search.toLowerCase())
    const matchStatus = statusFilter ? i.status === statusFilter : true
    return matchSearch && matchStatus
  })

  return (
    <div className="p-6 max-w-5xl">
      <h1 className="text-lg font-semibold text-zinc-100 mb-4">Issues</h1>
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
