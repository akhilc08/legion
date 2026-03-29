import { useState } from 'react'
import { useParams } from 'react-router-dom'
import { useQuery } from '@tanstack/react-query'
import { apiClient } from '@/lib/api'
import type { AuditLog } from '@/lib/types'

export function Audit() {
  const { companyId } = useParams<{ companyId: string }>()
  const [expanded, setExpanded] = useState<string | null>(null)

  const { data: logs = [] } = useQuery<AuditLog[]>({
    queryKey: ['audit', companyId],
    queryFn: () => apiClient.get(`/api/companies/${companyId}/audit?limit=100`).then((r) => r.data),
    enabled: !!companyId,
  })

  return (
    <div className="p-6 max-w-5xl">
      <h1 className="text-lg font-semibold text-zinc-100 mb-4">Audit Log</h1>
      <div className="rounded-md border border-zinc-800 overflow-hidden">
        <table className="w-full">
          <thead>
            <tr className="border-b border-zinc-800 bg-zinc-900">
              <th className="px-3 py-2 text-left text-xs font-medium text-zinc-500">Time</th>
              <th className="px-3 py-2 text-left text-xs font-medium text-zinc-500">Event</th>
              <th className="px-3 py-2 text-left text-xs font-medium text-zinc-500">Actor</th>
              <th className="px-3 py-2 text-left text-xs font-medium text-zinc-500">Payload</th>
            </tr>
          </thead>
          <tbody>
            {logs.map((log) => (
              <>
                <tr key={log.id} className="border-b border-zinc-800 hover:bg-zinc-900/50 cursor-pointer" onClick={() => setExpanded(expanded === log.id ? null : log.id)}>
                  <td className="px-3 py-2 text-xs text-zinc-500 whitespace-nowrap">{new Date(log.created_at).toLocaleString()}</td>
                  <td className="px-3 py-2 text-xs font-mono text-zinc-300">{log.event_type}</td>
                  <td className="px-3 py-2 text-xs text-zinc-500">{log.actor_id ? log.actor_id.slice(0, 8) + '…' : '—'}</td>
                  <td className="px-3 py-2 text-xs text-zinc-600">{expanded === log.id ? '▲ hide' : '▼ show'}</td>
                </tr>
                {expanded === log.id && (
                  <tr key={`${log.id}-exp`} className="border-b border-zinc-800 bg-zinc-900/30">
                    <td colSpan={4} className="px-4 py-3">
                      <pre className="text-xs text-zinc-400 whitespace-pre-wrap font-mono">{JSON.stringify(log.payload, null, 2)}</pre>
                    </td>
                  </tr>
                )}
              </>
            ))}
            {logs.length === 0 && <tr><td colSpan={4} className="px-3 py-6 text-center text-sm text-zinc-600">No events yet.</td></tr>}
          </tbody>
        </table>
      </div>
    </div>
  )
}
