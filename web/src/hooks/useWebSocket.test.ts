import { describe, it, expect, vi, beforeEach } from 'vitest'
import { patchCacheFromEvent } from './useWebSocket'
import type { WsEvent, Agent } from '@/lib/types'
import { QueryClient } from '@tanstack/react-query'

function makeAgent(overrides: Partial<Agent> = {}): Agent {
  return {
    id: 'a1',
    company_id: 'c1',
    role: 'engineer',
    title: 'Engineer',
    system_prompt: '',
    manager_id: null,
    runtime: 'claude_code',
    status: 'idle',
    monthly_budget: 100000,
    token_spend: 0,
    chat_token_spend: 0,
    pid: null,
    created_at: '2024-01-01T00:00:00Z',
    updated_at: '2024-01-01T00:00:00Z',
    ...overrides,
  }
}

describe('patchCacheFromEvent – agent_status', () => {
  let qc: QueryClient
  beforeEach(() => { qc = new QueryClient() })

  it('updates matching agent status in cache', () => {
    qc.setQueryData(['agents', 'c1'], [makeAgent({ id: 'a1', status: 'idle' })])
    patchCacheFromEvent(qc, { type: 'agent_status', company_id: 'c1', payload: { agent_id: 'a1', status: 'working' } })
    const agents = qc.getQueryData<Agent[]>(['agents', 'c1'])
    expect(agents?.[0].status).toBe('working')
  })

  it('does NOT update a non-matching agent', () => {
    qc.setQueryData(['agents', 'c1'], [makeAgent({ id: 'a1', status: 'idle' })])
    patchCacheFromEvent(qc, { type: 'agent_status', company_id: 'c1', payload: { agent_id: 'a99', status: 'working' } })
    const agents = qc.getQueryData<Agent[]>(['agents', 'c1'])
    expect(agents?.[0].status).toBe('idle')
  })

  it('handles undefined cache (no prior setQueryData) gracefully', () => {
    expect(() =>
      patchCacheFromEvent(qc, { type: 'agent_status', company_id: 'c1', payload: { agent_id: 'a1', status: 'working' } })
    ).not.toThrow()
    expect(qc.getQueryData(['agents', 'c1'])).toBeUndefined()
  })

  it('handles multiple agents in cache, only updates the correct one', () => {
    qc.setQueryData(['agents', 'c1'], [
      makeAgent({ id: 'a1', status: 'idle' }),
      makeAgent({ id: 'a2', status: 'idle' }),
      makeAgent({ id: 'a3', status: 'idle' }),
    ])
    patchCacheFromEvent(qc, { type: 'agent_status', company_id: 'c1', payload: { agent_id: 'a2', status: 'working' } })
    const agents = qc.getQueryData<Agent[]>(['agents', 'c1'])
    expect(agents?.[0].status).toBe('idle')
    expect(agents?.[1].status).toBe('working')
    expect(agents?.[2].status).toBe('idle')
  })

  it('status "idle" is applied correctly', () => {
    qc.setQueryData(['agents', 'c1'], [makeAgent({ id: 'a1', status: 'working' })])
    patchCacheFromEvent(qc, { type: 'agent_status', company_id: 'c1', payload: { agent_id: 'a1', status: 'idle' } })
    expect(qc.getQueryData<Agent[]>(['agents', 'c1'])?.[0].status).toBe('idle')
  })

  it('status "working" is applied correctly', () => {
    qc.setQueryData(['agents', 'c1'], [makeAgent({ id: 'a1', status: 'idle' })])
    patchCacheFromEvent(qc, { type: 'agent_status', company_id: 'c1', payload: { agent_id: 'a1', status: 'working' } })
    expect(qc.getQueryData<Agent[]>(['agents', 'c1'])?.[0].status).toBe('working')
  })

  it('status "paused" is applied correctly', () => {
    qc.setQueryData(['agents', 'c1'], [makeAgent({ id: 'a1', status: 'idle' })])
    patchCacheFromEvent(qc, { type: 'agent_status', company_id: 'c1', payload: { agent_id: 'a1', status: 'paused' } })
    expect(qc.getQueryData<Agent[]>(['agents', 'c1'])?.[0].status).toBe('paused')
  })

  it('status "blocked" is applied correctly', () => {
    qc.setQueryData(['agents', 'c1'], [makeAgent({ id: 'a1', status: 'idle' })])
    patchCacheFromEvent(qc, { type: 'agent_status', company_id: 'c1', payload: { agent_id: 'a1', status: 'blocked' } })
    expect(qc.getQueryData<Agent[]>(['agents', 'c1'])?.[0].status).toBe('blocked')
  })

  it('status "failed" is applied correctly', () => {
    qc.setQueryData(['agents', 'c1'], [makeAgent({ id: 'a1', status: 'idle' })])
    patchCacheFromEvent(qc, { type: 'agent_status', company_id: 'c1', payload: { agent_id: 'a1', status: 'failed' } })
    expect(qc.getQueryData<Agent[]>(['agents', 'c1'])?.[0].status).toBe('failed')
  })

  it('status "done" is applied correctly', () => {
    qc.setQueryData(['agents', 'c1'], [makeAgent({ id: 'a1', status: 'idle' })])
    patchCacheFromEvent(qc, { type: 'agent_status', company_id: 'c1', payload: { agent_id: 'a1', status: 'done' } })
    expect(qc.getQueryData<Agent[]>(['agents', 'c1'])?.[0].status).toBe('done')
  })

  it('status "degraded" is applied correctly', () => {
    qc.setQueryData(['agents', 'c1'], [makeAgent({ id: 'a1', status: 'idle' })])
    patchCacheFromEvent(qc, { type: 'agent_status', company_id: 'c1', payload: { agent_id: 'a1', status: 'degraded' } })
    expect(qc.getQueryData<Agent[]>(['agents', 'c1'])?.[0].status).toBe('degraded')
  })

  it('uses company_id from event as cache key, not a hardcoded value', () => {
    qc.setQueryData(['agents', 'company-X'], [makeAgent({ id: 'a1', company_id: 'company-X', status: 'idle' })])
    patchCacheFromEvent(qc, { type: 'agent_status', company_id: 'company-X', payload: { agent_id: 'a1', status: 'working' } })
    expect(qc.getQueryData<Agent[]>(['agents', 'company-X'])?.[0].status).toBe('working')
    // Different company untouched
    expect(qc.getQueryData<Agent[]>(['agents', 'company-Y'])).toBeUndefined()
  })

  it('different company IDs update different cache entries independently', () => {
    qc.setQueryData(['agents', 'c1'], [makeAgent({ id: 'a1', company_id: 'c1', status: 'idle' })])
    qc.setQueryData(['agents', 'c2'], [makeAgent({ id: 'a1', company_id: 'c2', status: 'idle' })])
    patchCacheFromEvent(qc, { type: 'agent_status', company_id: 'c1', payload: { agent_id: 'a1', status: 'working' } })
    expect(qc.getQueryData<Agent[]>(['agents', 'c1'])?.[0].status).toBe('working')
    expect(qc.getQueryData<Agent[]>(['agents', 'c2'])?.[0].status).toBe('idle')
  })
})

describe('patchCacheFromEvent – invalidation events', () => {
  let qc: QueryClient
  beforeEach(() => { qc = new QueryClient() })

  it('issue_update: calls invalidateQueries with [issues, companyId]', () => {
    const spy = vi.spyOn(qc, 'invalidateQueries')
    patchCacheFromEvent(qc, { type: 'issue_update', company_id: 'c1', payload: {} })
    expect(spy).toHaveBeenCalledWith({ queryKey: ['issues', 'c1'] })
  })

  it('issue_update: uses company_id from event', () => {
    const spy = vi.spyOn(qc, 'invalidateQueries')
    patchCacheFromEvent(qc, { type: 'issue_update', company_id: 'company-99', payload: {} })
    expect(spy).toHaveBeenCalledWith({ queryKey: ['issues', 'company-99'] })
  })

  it('notification: calls invalidateQueries with [notifications, companyId]', () => {
    const spy = vi.spyOn(qc, 'invalidateQueries')
    patchCacheFromEvent(qc, { type: 'notification', company_id: 'c1', payload: {} })
    expect(spy).toHaveBeenCalledWith({ queryKey: ['notifications', 'c1'] })
  })

  it('notification: uses company_id from event', () => {
    const spy = vi.spyOn(qc, 'invalidateQueries')
    patchCacheFromEvent(qc, { type: 'notification', company_id: 'cx', payload: {} })
    expect(spy).toHaveBeenCalledWith({ queryKey: ['notifications', 'cx'] })
  })

  it('hire_pending: calls invalidateQueries with [hires, companyId]', () => {
    const spy = vi.spyOn(qc, 'invalidateQueries')
    patchCacheFromEvent(qc, { type: 'hire_pending', company_id: 'c1', payload: {} })
    expect(spy).toHaveBeenCalledWith({ queryKey: ['hires', 'c1'] })
  })

  it('escalation: calls invalidateQueries with [notifications, companyId]', () => {
    const spy = vi.spyOn(qc, 'invalidateQueries')
    patchCacheFromEvent(qc, { type: 'escalation', company_id: 'c1', payload: {} })
    expect(spy).toHaveBeenCalledWith({ queryKey: ['notifications', 'c1'] })
  })

  it('agent_hired: invalidates [agents, companyId]', () => {
    const spy = vi.spyOn(qc, 'invalidateQueries')
    patchCacheFromEvent(qc, { type: 'agent_hired', company_id: 'c1', payload: {} })
    expect(spy).toHaveBeenCalledWith({ queryKey: ['agents', 'c1'] })
  })

  it('agent_hired: also invalidates [hires, companyId]', () => {
    const spy = vi.spyOn(qc, 'invalidateQueries')
    patchCacheFromEvent(qc, { type: 'agent_hired', company_id: 'c1', payload: {} })
    expect(spy).toHaveBeenCalledWith({ queryKey: ['hires', 'c1'] })
  })

  it('agent_hired: invalidates both agents and hires (total 2 calls)', () => {
    const spy = vi.spyOn(qc, 'invalidateQueries')
    patchCacheFromEvent(qc, { type: 'agent_hired', company_id: 'c1', payload: {} })
    expect(spy).toHaveBeenCalledTimes(2)
  })

  it('agent_approved: invalidates [agents, companyId]', () => {
    const spy = vi.spyOn(qc, 'invalidateQueries')
    patchCacheFromEvent(qc, { type: 'agent_approved', company_id: 'c1', payload: {} })
    expect(spy).toHaveBeenCalledWith({ queryKey: ['agents', 'c1'] })
  })

  it('agent_approved: also invalidates [hires, companyId]', () => {
    const spy = vi.spyOn(qc, 'invalidateQueries')
    patchCacheFromEvent(qc, { type: 'agent_approved', company_id: 'c1', payload: {} })
    expect(spy).toHaveBeenCalledWith({ queryKey: ['hires', 'c1'] })
  })

  it('agent_approved: invalidates both agents and hires (total 2 calls)', () => {
    const spy = vi.spyOn(qc, 'invalidateQueries')
    patchCacheFromEvent(qc, { type: 'agent_approved', company_id: 'c1', payload: {} })
    expect(spy).toHaveBeenCalledTimes(2)
  })
})

describe('patchCacheFromEvent – no-op event types', () => {
  let qc: QueryClient
  beforeEach(() => { qc = new QueryClient() })

  it('agent_log: does nothing, no error', () => {
    const spy = vi.spyOn(qc, 'invalidateQueries')
    expect(() =>
      patchCacheFromEvent(qc, { type: 'agent_log', company_id: 'c1', payload: { message: 'log line' } })
    ).not.toThrow()
    expect(spy).not.toHaveBeenCalled()
  })

  it('heartbeat: does nothing, no error', () => {
    const spy = vi.spyOn(qc, 'invalidateQueries')
    expect(() =>
      patchCacheFromEvent(qc, { type: 'heartbeat', company_id: 'c1', payload: {} })
    ).not.toThrow()
    expect(spy).not.toHaveBeenCalled()
  })

  it('runtime_status: does nothing, no error', () => {
    const spy = vi.spyOn(qc, 'invalidateQueries')
    expect(() =>
      patchCacheFromEvent(qc, { type: 'runtime_status', company_id: 'c1', payload: {} })
    ).not.toThrow()
    expect(spy).not.toHaveBeenCalled()
  })

  it('chat_message: does nothing, no error', () => {
    const spy = vi.spyOn(qc, 'invalidateQueries')
    expect(() =>
      patchCacheFromEvent(qc, { type: 'chat_message', company_id: 'c1', payload: {} })
    ).not.toThrow()
    expect(spy).not.toHaveBeenCalled()
  })

  it('unknown event type cast: does not throw', () => {
    const spy = vi.spyOn(qc, 'invalidateQueries')
    expect(() =>
      patchCacheFromEvent(qc, { type: 'unknown_future_event' as WsEvent['type'], company_id: 'c1', payload: {} })
    ).not.toThrow()
    expect(spy).not.toHaveBeenCalled()
  })
})
