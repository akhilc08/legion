import { useEffect, useRef } from 'react'
import { useQueryClient, type QueryClient } from '@tanstack/react-query'
import type { WsEvent, Agent } from '@/lib/types'

export function patchCacheFromEvent(qc: QueryClient, event: WsEvent) {
  const companyId = event.company_id
  switch (event.type) {
    case 'agent_status': {
      const payload = event.payload as { agent_id: string; status: Agent['status'] }
      qc.setQueryData<Agent[]>(['agents', companyId], (old) =>
        old?.map((a) => a.id === payload.agent_id ? { ...a, status: payload.status } : a)
      )
      break
    }
    case 'issue_update':
      qc.invalidateQueries({ queryKey: ['issues', companyId] })
      break
    case 'notification':
      qc.invalidateQueries({ queryKey: ['notifications', companyId] })
      break
    case 'hire_pending':
      qc.invalidateQueries({ queryKey: ['hires', companyId] })
      break
    case 'escalation':
      qc.invalidateQueries({ queryKey: ['notifications', companyId] })
      break
    case 'agent_hired':
    case 'agent_approved':
      qc.invalidateQueries({ queryKey: ['agents', companyId] })
      qc.invalidateQueries({ queryKey: ['hires', companyId] })
      break
    default:
      break
  }
}

export function useWebSocket(companyId: string | null) {
  const qc = useQueryClient()
  const wsRef = useRef<WebSocket | null>(null)

  useEffect(() => {
    if (!companyId) return
    const token = localStorage.getItem('legion_token')
    if (!token) return
    const protocol = window.location.protocol === 'https:' ? 'wss' : 'ws'
    const host = window.location.host
    const url = `${protocol}://${host}/api/companies/${companyId}/ws?token=${token}`
    const ws = new WebSocket(url)
    wsRef.current = ws
    ws.onmessage = (e) => {
      try {
        const event: WsEvent = JSON.parse(e.data)
        patchCacheFromEvent(qc, event)
      } catch { /* ignore malformed */ }
    }
    return () => { ws.close(); wsRef.current = null }
  }, [companyId, qc])
}
