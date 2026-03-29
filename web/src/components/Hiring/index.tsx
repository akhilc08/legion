import { useParams } from 'react-router-dom'
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { apiClient } from '@/lib/api'
import type { PendingHire } from '@/lib/types'
import { HireCard } from './HireCard'

export function Hiring() {
  const { companyId } = useParams<{ companyId: string }>()
  const qc = useQueryClient()

  const { data: hires = [] } = useQuery<PendingHire[]>({
    queryKey: ['hires', companyId],
    queryFn: () => apiClient.get(`/api/companies/${companyId}/hires`).then((r) => r.data),
    enabled: !!companyId,
  })

  const approve = useMutation({
    mutationFn: (hireId: string) => apiClient.post(`/api/companies/${companyId}/hires/${hireId}/approve`),
    onSuccess: () => qc.invalidateQueries({ queryKey: ['hires', companyId] }),
  })
  const reject = useMutation({
    mutationFn: (hireId: string) => apiClient.post(`/api/companies/${companyId}/hires/${hireId}/reject`),
    onSuccess: () => qc.invalidateQueries({ queryKey: ['hires', companyId] }),
  })

  const pending = hires.filter((h) => h.status === 'pending')

  return (
    <div className="p-6 max-w-3xl">
      <h1 className="text-lg font-semibold text-zinc-100 mb-4">
        Hiring{pending.length > 0 && <span className="ml-2 text-zinc-500 text-sm font-normal">({pending.length} pending)</span>}
      </h1>
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
