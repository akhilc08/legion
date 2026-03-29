import { describe, it, expect, beforeEach } from 'vitest'
import { useAppStore } from './useAppStore'

beforeEach(() => {
  useAppStore.setState({ token: null, companyId: null, agentId: null })
  localStorage.clear()
})

describe('useAppStore', () => {
  it('setToken persists to localStorage and updates state', () => {
    useAppStore.getState().setToken('abc123')
    expect(useAppStore.getState().token).toBe('abc123')
    expect(localStorage.getItem('legion_token')).toBe('abc123')
  })

  it('setToken(null) removes from localStorage', () => {
    useAppStore.getState().setToken('abc123')
    useAppStore.getState().setToken(null)
    expect(useAppStore.getState().token).toBeNull()
    expect(localStorage.getItem('legion_token')).toBeNull()
  })

  it('initialises token from localStorage', () => {
    localStorage.setItem('legion_token', 'persisted')
    // Re-create store by importing fresh state via getInitialToken
    expect(useAppStore.getState().getInitialToken()).toBe('persisted')
  })

  it('setCompanyId updates selected company', () => {
    useAppStore.getState().setCompanyId('company-abc')
    expect(useAppStore.getState().companyId).toBe('company-abc')
  })

  it('setAgentId updates selected agent', () => {
    useAppStore.getState().setAgentId('agent-xyz')
    expect(useAppStore.getState().agentId).toBe('agent-xyz')
  })
})
