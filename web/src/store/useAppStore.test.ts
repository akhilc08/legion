import { describe, it, expect, beforeEach } from 'vitest'
import { useAppStore } from './useAppStore'

beforeEach(() => {
  localStorage.clear()
  useAppStore.setState({ token: null, companyId: null, agentId: null })
})

describe('useAppStore – setToken', () => {
  it('setToken stores value in state', () => {
    useAppStore.getState().setToken('abc123')
    expect(useAppStore.getState().token).toBe('abc123')
  })

  it('setToken persists value to localStorage', () => {
    useAppStore.getState().setToken('abc123')
    expect(localStorage.getItem('legion_token')).toBe('abc123')
  })

  it('setToken(null) sets state to null', () => {
    useAppStore.getState().setToken('abc123')
    useAppStore.getState().setToken(null)
    expect(useAppStore.getState().token).toBeNull()
  })

  it('setToken(null) removes from localStorage', () => {
    useAppStore.getState().setToken('abc123')
    useAppStore.getState().setToken(null)
    expect(localStorage.getItem('legion_token')).toBeNull()
  })

  it('setToken with empty string sets state to empty string (falsy branch)', () => {
    // Empty string is falsy in JS so setToken('') goes to the else branch: removeItem
    useAppStore.getState().setToken('')
    expect(useAppStore.getState().token).toBe('')
    // localStorage.removeItem was called because '' is falsy
    expect(localStorage.getItem('legion_token')).toBeNull()
  })

  it('multiple calls to setToken overwrite correctly', () => {
    useAppStore.getState().setToken('first')
    useAppStore.getState().setToken('second')
    useAppStore.getState().setToken('third')
    expect(useAppStore.getState().token).toBe('third')
    expect(localStorage.getItem('legion_token')).toBe('third')
  })

  it('setToken then setToken(null) leaves localStorage null', () => {
    useAppStore.getState().setToken('to-remove')
    useAppStore.getState().setToken(null)
    expect(localStorage.getItem('legion_token')).toBeNull()
  })

  it('setToken with a JWT-shaped string works', () => {
    const jwt = 'eyJhbGci.payload.sig'
    useAppStore.getState().setToken(jwt)
    expect(useAppStore.getState().token).toBe(jwt)
    expect(localStorage.getItem('legion_token')).toBe(jwt)
  })

  it('setToken does not affect companyId', () => {
    useAppStore.getState().setCompanyId('company-1')
    useAppStore.getState().setToken('tok')
    expect(useAppStore.getState().companyId).toBe('company-1')
  })

  it('setToken does not affect agentId', () => {
    useAppStore.getState().setAgentId('agent-1')
    useAppStore.getState().setToken('tok')
    expect(useAppStore.getState().agentId).toBe('agent-1')
  })
})

describe('useAppStore – setCompanyId', () => {
  it('setCompanyId sets companyId in state', () => {
    useAppStore.getState().setCompanyId('company-abc')
    expect(useAppStore.getState().companyId).toBe('company-abc')
  })

  it('setCompanyId(null) sets companyId to null', () => {
    useAppStore.getState().setCompanyId('company-abc')
    useAppStore.getState().setCompanyId(null)
    expect(useAppStore.getState().companyId).toBeNull()
  })

  it('overwriting companyId works', () => {
    useAppStore.getState().setCompanyId('c1')
    useAppStore.getState().setCompanyId('c2')
    expect(useAppStore.getState().companyId).toBe('c2')
  })

  it('setCompanyId does not affect token', () => {
    useAppStore.getState().setToken('tok')
    useAppStore.getState().setCompanyId('c1')
    expect(useAppStore.getState().token).toBe('tok')
  })

  it('setCompanyId does not affect agentId', () => {
    useAppStore.getState().setAgentId('a1')
    useAppStore.getState().setCompanyId('c1')
    expect(useAppStore.getState().agentId).toBe('a1')
  })
})

describe('useAppStore – setAgentId', () => {
  it('setAgentId sets agentId in state', () => {
    useAppStore.getState().setAgentId('agent-xyz')
    expect(useAppStore.getState().agentId).toBe('agent-xyz')
  })

  it('setAgentId(null) sets agentId to null', () => {
    useAppStore.getState().setAgentId('agent-xyz')
    useAppStore.getState().setAgentId(null)
    expect(useAppStore.getState().agentId).toBeNull()
  })

  it('overwriting agentId works', () => {
    useAppStore.getState().setAgentId('a1')
    useAppStore.getState().setAgentId('a2')
    expect(useAppStore.getState().agentId).toBe('a2')
  })

  it('setAgentId does not affect token', () => {
    useAppStore.getState().setToken('tok')
    useAppStore.getState().setAgentId('a1')
    expect(useAppStore.getState().token).toBe('tok')
  })

  it('setAgentId does not affect companyId', () => {
    useAppStore.getState().setCompanyId('c1')
    useAppStore.getState().setAgentId('a1')
    expect(useAppStore.getState().companyId).toBe('c1')
  })
})

describe('useAppStore – getInitialToken', () => {
  it('returns value from localStorage when key is present', () => {
    localStorage.setItem('legion_token', 'persisted-tok')
    expect(useAppStore.getState().getInitialToken()).toBe('persisted-tok')
  })

  it('returns null when localStorage is empty', () => {
    expect(useAppStore.getState().getInitialToken()).toBeNull()
  })

  it('returns updated value after setToken call', () => {
    useAppStore.getState().setToken('new-tok')
    expect(useAppStore.getState().getInitialToken()).toBe('new-tok')
  })

  it('returns null after setToken(null)', () => {
    useAppStore.getState().setToken('tok')
    useAppStore.getState().setToken(null)
    expect(useAppStore.getState().getInitialToken()).toBeNull()
  })
})

describe('useAppStore – initial state', () => {
  it('companyId starts as null', () => {
    expect(useAppStore.getState().companyId).toBeNull()
  })

  it('agentId starts as null', () => {
    expect(useAppStore.getState().agentId).toBeNull()
  })

  it('token starts as null when localStorage is empty', () => {
    // Already cleared in beforeEach and state reset
    expect(useAppStore.getState().token).toBeNull()
  })

  it('state is isolated between tests (beforeEach clears)', () => {
    // Set values
    useAppStore.getState().setToken('test')
    useAppStore.getState().setCompanyId('c')
    useAppStore.getState().setAgentId('a')
    // Simulate what beforeEach does
    localStorage.clear()
    useAppStore.setState({ token: null, companyId: null, agentId: null })
    expect(useAppStore.getState().token).toBeNull()
    expect(useAppStore.getState().companyId).toBeNull()
    expect(useAppStore.getState().agentId).toBeNull()
  })
})
