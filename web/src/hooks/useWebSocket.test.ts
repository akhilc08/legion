import { describe, it, expect, vi, beforeEach } from 'vitest'
import { patchCacheFromEvent } from './useWebSocket'
import type { WsEvent, Agent } from '@/lib/types'
import { QueryClient } from '@tanstack/react-query'

describe('patchCacheFromEvent', () => {
  let qc: QueryClient
  beforeEach(() => { qc = new QueryClient() })

  it('updates agent status in the agents cache on agent_status event', () => {
    const existingAgent: Agent = {
      id: 'a1', company_id: 'c1', role: 'cto', title: 'CTO',
      system_prompt: '', manager_id: null, runtime: 'claude_code',
      status: 'idle', monthly_budget: 100000, token_spend: 0,
      chat_token_spend: 0, pid: null, created_at: '', updated_at: '',
    }
    qc.setQueryData(['agents', 'c1'], [existingAgent])
    patchCacheFromEvent(qc, { type: 'agent_status', company_id: 'c1', payload: { agent_id: 'a1', status: 'working' } })
    const updated = qc.getQueryData<Agent[]>(['agents', 'c1'])
    expect(updated?.[0].status).toBe('working')
  })

  it('invalidates notifications cache on notification event', () => {
    const invalidate = vi.spyOn(qc, 'invalidateQueries')
    patchCacheFromEvent(qc, { type: 'notification', company_id: 'c1', payload: {} })
    expect(invalidate).toHaveBeenCalledWith({ queryKey: ['notifications', 'c1'] })
  })

  it('invalidates hires cache on hire_pending event', () => {
    const invalidate = vi.spyOn(qc, 'invalidateQueries')
    patchCacheFromEvent(qc, { type: 'hire_pending', company_id: 'c1', payload: {} })
    expect(invalidate).toHaveBeenCalledWith({ queryKey: ['hires', 'c1'] })
  })
})
