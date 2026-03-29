import { useParams } from 'react-router-dom'
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { apiClient } from '@/lib/api'
import type { Agent, Notification } from '@/lib/types'
import { AgentStatusCard } from './AgentStatusCard'
import { EscalationList } from './EscalationList'

const STATUS_CARDS = [
  { label: 'Working', key: 'working' as const, colour: 'text-emerald-400' },
  { label: 'Idle', key: 'idle' as const, colour: 'text-zinc-300' },
  { label: 'Blocked', key: 'blocked' as const, colour: 'text-amber-400' },
  { label: 'Failed', key: 'failed' as const, colour: 'text-red-400' },
  { label: 'Degraded', key: 'degraded' as const, colour: 'text-orange-400' },
]

export function Dashboard() {
  const { companyId } = useParams<{ companyId: string }>()
  const qc = useQueryClient()

  const { data: agents = [] } = useQuery<Agent[]>({
    queryKey: ['agents', companyId],
    queryFn: () => apiClient.get(`/api/companies/${companyId}/agents`).then((r) => r.data),
    enabled: !!companyId,
  })

  const { data: notifications = [] } = useQuery<Notification[]>({
    queryKey: ['notifications', companyId],
    queryFn: () =>
      apiClient.get(`/api/companies/${companyId}/notifications`).then((r) => r.data),
    enabled: !!companyId,
  })

  const dismiss = useMutation({
    mutationFn: (id: string) =>
      apiClient.post(`/api/companies/${companyId}/notifications/${id}/dismiss`),
    onSuccess: () => qc.invalidateQueries({ queryKey: ['notifications', companyId] }),
  })

  const countByStatus = (status: Agent['status']) =>
    agents.filter((a) => a.status === status).length

  return (
    <div className="p-6 space-y-6 max-w-4xl">
      <h1 className="text-lg font-semibold text-zinc-100">Dashboard</h1>

      <div className="grid grid-cols-5 gap-3">
        {STATUS_CARDS.map(({ label, key, colour }) => (
          <AgentStatusCard
            key={key}
            label={label}
            count={countByStatus(key)}
            colour={colour}
          />
        ))}
      </div>

      <div>
        <h2 className="text-sm font-medium text-zinc-400 mb-3">Notifications</h2>
        <EscalationList
          notifications={notifications}
          onDismiss={(id) => dismiss.mutate(id)}
        />
      </div>
    </div>
  )
}
