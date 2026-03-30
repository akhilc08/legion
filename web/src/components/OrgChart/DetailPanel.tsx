import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { apiClient } from '@/lib/api'
import { useAppStore } from '@/store/useAppStore'
import type { Agent, Issue } from '@/lib/types'
import { Button } from '@/components/ui/button'
import { X } from 'lucide-react'
import { cn } from '@/lib/utils'

const STATUS_COLOURS: Record<Agent['status'], string> = {
  idle: 'text-zinc-400', working: 'text-emerald-400', paused: 'text-blue-400',
  blocked: 'text-amber-400', failed: 'text-red-400', done: 'text-zinc-500', degraded: 'text-orange-400',
}

interface DetailPanelProps { agent: Agent; onClose: () => void }

export function DetailPanel({ agent, onClose }: DetailPanelProps) {
  const companyId = useAppStore((s) => s.companyId)
  const qc = useQueryClient()

  const { data: issues = [] } = useQuery<Issue[]>({
    queryKey: ['issues', companyId],
    queryFn: () => apiClient.get(`/api/companies/${companyId}/issues`).then((r) => r.data),
    enabled: !!companyId,
  })

  const currentIssue = issues.find((i) => i.assignee_id === agent.id && i.status === 'in_progress')

  const spawn = useMutation({
    mutationFn: () => apiClient.post(`/api/companies/${companyId}/agents/${agent.id}/spawn`),
    onSuccess: () => qc.invalidateQueries({ queryKey: ['agents', companyId] }),
  })
  const kill = useMutation({
    mutationFn: () => apiClient.post(`/api/companies/${companyId}/agents/${agent.id}/kill`),
    onSuccess: () => qc.invalidateQueries({ queryKey: ['agents', companyId] }),
  })
  const pause = useMutation({
    mutationFn: () => apiClient.post(`/api/companies/${companyId}/agents/${agent.id}/${agent.status === 'paused' ? 'resume' : 'pause'}`),
    onSuccess: () => qc.invalidateQueries({ queryKey: ['agents', companyId] }),
  })

  const unlimited = agent.monthly_budget === 0 || agent.monthly_budget >= 2147483647
  const budgetPct = unlimited ? 0 : Math.min(100, Math.round((agent.token_spend / agent.monthly_budget) * 100))

  return (
    <div className="w-72 border-l border-zinc-800 bg-zinc-950 p-4 flex flex-col gap-4 overflow-y-auto">
      <div className="flex items-center justify-between">
        <h3 className="text-sm font-semibold text-zinc-100">{agent.title}</h3>
        <button onClick={onClose} className="text-zinc-600 hover:text-zinc-300"><X className="h-4 w-4" /></button>
      </div>
      <div className="flex items-center gap-2">
        <span className={cn('text-xs font-medium', STATUS_COLOURS[agent.status])}>● {agent.status}</span>
        <span className="text-xs text-zinc-600">{agent.runtime}</span>
      </div>
      {currentIssue && (
        <div className="rounded-md border border-zinc-800 bg-zinc-900 px-3 py-2">
          <p className="text-xs text-zinc-500 mb-0.5">Current issue</p>
          <p className="text-xs text-zinc-300 line-clamp-2">{currentIssue.title}</p>
          <p className="text-xs text-zinc-600 mt-1">attempt #{currentIssue.attempt_count}</p>
        </div>
      )}
      <div>
        <div className="flex justify-between text-xs text-zinc-500 mb-1">
          <span>Token budget</span><span>{unlimited ? '∞' : `${budgetPct}%`}</span>
        </div>
        <div className="h-1.5 rounded-full bg-zinc-800">
          <div className={cn('h-1.5 rounded-full', budgetPct > 80 ? 'bg-red-500' : budgetPct > 50 ? 'bg-amber-500' : 'bg-emerald-500')} style={{ width: `${budgetPct}%` }} />
        </div>
        <p className="text-xs text-zinc-600 mt-1">{agent.token_spend.toLocaleString()} / {unlimited ? '∞' : agent.monthly_budget.toLocaleString()} tokens</p>
      </div>
      <div className="space-y-2">
        {agent.status === 'idle' && <Button size="sm" className="w-full" disabled={spawn.isPending} onClick={() => spawn.mutate()}>Spawn</Button>}
        {(agent.status === 'working' || agent.status === 'paused') && (
          <Button size="sm" variant="secondary" className="w-full" disabled={pause.isPending} onClick={() => pause.mutate()}>
            {agent.status === 'paused' ? 'Resume' : 'Pause'}
          </Button>
        )}
        {agent.status !== 'idle' && agent.status !== 'done' && (
          <Button size="sm" variant="destructive" className="w-full" disabled={kill.isPending} onClick={() => kill.mutate()}>Kill</Button>
        )}
      </div>
    </div>
  )
}
