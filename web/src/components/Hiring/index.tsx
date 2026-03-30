import { useState } from 'react'
import { useParams } from 'react-router-dom'
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { apiClient } from '@/lib/api'
import type { Agent, PendingHire } from '@/lib/types'
import { HireCard } from './HireCard'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'

const RUNTIMES = ['claude_code', 'openclaw'] as const

export function Hiring() {
  const { companyId } = useParams<{ companyId: string }>()
  const qc = useQueryClient()
  const [showForm, setShowForm] = useState(false)
  const [form, setForm] = useState({
    role_title: '',
    system_prompt: '',
    runtime: 'claude_code',
    budget_allocation: '',
    reporting_to_agent_id: '',
    requested_by_agent_id: '',
    initial_task: '',
  })

  const { data: hires = [] } = useQuery<PendingHire[]>({
    queryKey: ['hires', companyId],
    queryFn: () => apiClient.get(`/api/companies/${companyId}/hires`).then((r) => r.data),
    enabled: !!companyId,
  })
  const { data: rawAgents = [] } = useQuery<Agent[]>({
    queryKey: ['agents', companyId],
    queryFn: () => apiClient.get(`/api/companies/${companyId}/agents`).then((r) => r.data),
    enabled: !!companyId,
  })
  // Board always first
  const agents = [...rawAgents].sort((a, b) =>
    a.role === 'board' ? -1 : b.role === 'board' ? 1 : 0
  )

  const approve = useMutation({
    mutationFn: (hireId: string) => apiClient.post(`/api/companies/${companyId}/hires/${hireId}/approve`),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ['hires', companyId] })
      qc.invalidateQueries({ queryKey: ['agents', companyId] })
    },
  })
  const reject = useMutation({
    mutationFn: (hireId: string) => apiClient.post(`/api/companies/${companyId}/hires/${hireId}/reject`),
    onSuccess: () => qc.invalidateQueries({ queryKey: ['hires', companyId] }),
  })
  const create = useMutation({
    mutationFn: () =>
      apiClient.post(`/api/companies/${companyId}/hires`, {
        role_title: form.role_title,
        system_prompt: form.system_prompt,
        runtime: form.runtime,
        budget_allocation: form.budget_allocation.trim() === '' ? 0 : parseInt(form.budget_allocation, 10) || 0,
        reporting_to_agent_id: form.reporting_to_agent_id,
        requested_by_agent_id: form.requested_by_agent_id,
        initial_task: form.initial_task || null,
      }),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ['hires', companyId] })
      setForm({ role_title: '', system_prompt: '', runtime: 'claude_code', budget_allocation: '', reporting_to_agent_id: '', requested_by_agent_id: '', initial_task: '' })
      setShowForm(false)
    },
  })

  const pending = (Array.isArray(hires) ? hires : []).filter((h) => h.status === 'pending')

  return (
    <div className="p-6 max-w-3xl">
      <div className="flex items-center justify-between mb-4">
        <h1 className="text-lg font-semibold text-zinc-100">
          Hiring{pending.length > 0 && <span className="ml-2 text-zinc-500 text-sm font-normal">({pending.length} pending)</span>}
        </h1>
        <Button size="sm" onClick={() => setShowForm((v) => !v)} variant={showForm ? 'secondary' : 'default'}>
          {showForm ? 'Cancel' : 'Request Hire'}
        </Button>
      </div>

      {showForm && (
        <div className="mb-6 rounded-md border border-zinc-700 bg-zinc-900 p-4 space-y-3">
          <div className="grid grid-cols-2 gap-3">
            <div className="col-span-2">
              <Input
                placeholder="Role title"
                value={form.role_title}
                onChange={(e) => setForm((f) => ({ ...f, role_title: e.target.value }))}
                className="bg-zinc-800 border-zinc-700 text-zinc-100"
              />
            </div>
            <select
              value={form.requested_by_agent_id}
              onChange={(e) => setForm((f) => ({ ...f, requested_by_agent_id: e.target.value }))}
              className="rounded-md border border-zinc-700 bg-zinc-800 text-zinc-300 text-sm px-3 py-2"
            >
              <option value="">Requested by…</option>
              {agents.map((a) => <option key={a.id} value={a.id}>{a.title}</option>)}
            </select>
            <select
              value={form.reporting_to_agent_id}
              onChange={(e) => setForm((f) => ({ ...f, reporting_to_agent_id: e.target.value }))}
              className="rounded-md border border-zinc-700 bg-zinc-800 text-zinc-300 text-sm px-3 py-2"
            >
              <option value="">Reports to…</option>
              {agents.map((a) => <option key={a.id} value={a.id}>{a.title}</option>)}
            </select>
            <select
              value={form.runtime}
              onChange={(e) => setForm((f) => ({ ...f, runtime: e.target.value }))}
              className="rounded-md border border-zinc-700 bg-zinc-800 text-zinc-300 text-sm px-3 py-2"
            >
              {RUNTIMES.map((r) => <option key={r} value={r}>{r}</option>)}
            </select>
            <Input
              type="number"
              placeholder="Monthly budget (empty = unlimited)"
              value={form.budget_allocation}
              onChange={(e) => setForm((f) => ({ ...f, budget_allocation: e.target.value }))}
              className="bg-zinc-800 border-zinc-700 text-zinc-100"
            />
          </div>
          <textarea
            placeholder="System prompt"
            value={form.system_prompt}
            onChange={(e) => setForm((f) => ({ ...f, system_prompt: e.target.value }))}
            rows={3}
            className="w-full rounded-md border border-zinc-700 bg-zinc-800 text-zinc-100 text-sm px-3 py-2 resize-none placeholder:text-zinc-500 focus:outline-none focus:ring-1 focus:ring-zinc-500"
          />
          <Input
            placeholder="Initial task (optional)"
            value={form.initial_task}
            onChange={(e) => setForm((f) => ({ ...f, initial_task: e.target.value }))}
            className="bg-zinc-800 border-zinc-700 text-zinc-100"
          />
          <div className="flex justify-end">
            <Button
              size="sm"
              disabled={
                !form.role_title.trim() ||
                !form.requested_by_agent_id ||
                !form.reporting_to_agent_id ||
                create.isPending
              }
              onClick={() => create.mutate()}
            >
              {create.isPending ? 'Submitting…' : 'Submit Request'}
            </Button>
          </div>
          {create.isError && (
            <p className="text-xs text-red-400">{String((create.error as Error).message)}</p>
          )}
        </div>
      )}

      {pending.length === 0 ? (
        <p className="text-sm text-zinc-600">No pending hire requests.</p>
      ) : (
        <div className="grid grid-cols-1 sm:grid-cols-2 gap-4">
          {pending.map((hire) => (
            <HireCard key={hire.id} hire={hire}
              onApprove={() => approve.mutate(hire.id)}
              onReject={() => reject.mutate(hire.id)}
              isApproving={approve.isPending && approve.variables === hire.id}
              isRejecting={reject.isPending && reject.variables === hire.id}
            />
          ))}
        </div>
      )}
    </div>
  )
}
