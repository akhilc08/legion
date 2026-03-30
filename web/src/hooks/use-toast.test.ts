import { describe, it, expect, vi } from 'vitest'
import { renderHook, act } from '@testing-library/react'
import { reducer, useToast, toast } from './use-toast'

// ---------------------------------------------------------------------------
// reducer – unit tests (pure function, no side effects we care about here)
// ---------------------------------------------------------------------------

describe('reducer – ADD_TOAST', () => {
  it('adds a toast to an empty state', () => {
    const state = { toasts: [] }
    const next = reducer(state, {
      type: 'ADD_TOAST',
      toast: { id: '1', title: 'Hello', open: true },
    })
    expect(next.toasts).toHaveLength(1)
    expect(next.toasts[0].id).toBe('1')
    expect(next.toasts[0].title).toBe('Hello')
  })

  it('adds new toast to front of list', () => {
    const state = { toasts: [{ id: '1', title: 'First', open: true }] }
    const next = reducer(state, {
      type: 'ADD_TOAST',
      toast: { id: '2', title: 'Second', open: true },
    })
    expect(next.toasts[0].id).toBe('2')
  })

  it('limits to TOAST_LIMIT of 1 – drops old toast when new one added', () => {
    const state = { toasts: [{ id: '1', title: 'Old', open: true }] }
    const next = reducer(state, {
      type: 'ADD_TOAST',
      toast: { id: '2', title: 'New', open: true },
    })
    expect(next.toasts).toHaveLength(1)
    expect(next.toasts[0].id).toBe('2')
  })

  it('does not mutate original state', () => {
    const state = { toasts: [] }
    reducer(state, { type: 'ADD_TOAST', toast: { id: '1', open: true } })
    expect(state.toasts).toHaveLength(0)
  })
})

describe('reducer – UPDATE_TOAST', () => {
  it('updates matching toast by id', () => {
    const state = { toasts: [{ id: '1', title: 'Old', open: true }] }
    const next = reducer(state, {
      type: 'UPDATE_TOAST',
      toast: { id: '1', title: 'Updated' },
    })
    expect(next.toasts[0].title).toBe('Updated')
    expect(next.toasts[0].open).toBe(true) // preserved
  })

  it('ignores toasts with non-matching id', () => {
    const state = {
      toasts: [
        { id: '1', title: 'First', open: true },
        { id: '2', title: 'Second', open: true },
      ],
    }
    const next = reducer(state, {
      type: 'UPDATE_TOAST',
      toast: { id: '99', title: 'Ghost' },
    })
    expect(next.toasts[0].title).toBe('First')
    expect(next.toasts[1].title).toBe('Second')
  })

  it('returns same length array after update', () => {
    const state = { toasts: [{ id: '1', title: 'T', open: true }] }
    const next = reducer(state, {
      type: 'UPDATE_TOAST',
      toast: { id: '1', title: 'T2' },
    })
    expect(next.toasts).toHaveLength(1)
  })
})

describe('reducer – DISMISS_TOAST', () => {
  it('sets open=false for given toastId', () => {
    const state = { toasts: [{ id: '1', title: 'T', open: true }] }
    const next = reducer(state, { type: 'DISMISS_TOAST', toastId: '1' })
    expect(next.toasts[0].open).toBe(false)
  })

  it('does not affect other toasts when dismissing by id', () => {
    const state = {
      toasts: [
        { id: '1', open: true },
        { id: '2', open: true },
      ],
    }
    const next = reducer(state, { type: 'DISMISS_TOAST', toastId: '1' })
    expect(next.toasts[0].open).toBe(false)
    expect(next.toasts[1].open).toBe(true)
  })

  it('sets open=false for all toasts when toastId is undefined', () => {
    const state = {
      toasts: [
        { id: '1', open: true },
        { id: '2', open: true },
      ],
    }
    const next = reducer(state, { type: 'DISMISS_TOAST', toastId: undefined })
    expect(next.toasts[0].open).toBe(false)
    expect(next.toasts[1].open).toBe(false)
  })

  it('keeps list length the same after dismiss', () => {
    const state = { toasts: [{ id: '1', open: true }] }
    const next = reducer(state, { type: 'DISMISS_TOAST', toastId: '1' })
    expect(next.toasts).toHaveLength(1)
  })
})

describe('reducer – REMOVE_TOAST', () => {
  it('removes toast with given id', () => {
    const state = { toasts: [{ id: '1', title: 'T', open: true }] }
    const next = reducer(state, { type: 'REMOVE_TOAST', toastId: '1' })
    expect(next.toasts).toHaveLength(0)
  })

  it('does not remove toasts with different id', () => {
    const state = {
      toasts: [
        { id: '1', open: true },
        { id: '2', open: true },
      ],
    }
    const next = reducer(state, { type: 'REMOVE_TOAST', toastId: '1' })
    expect(next.toasts).toHaveLength(1)
    expect(next.toasts[0].id).toBe('2')
  })

  it('removes all toasts when toastId is undefined', () => {
    const state = {
      toasts: [
        { id: '1', open: true },
        { id: '2', open: true },
      ],
    }
    const next = reducer(state, { type: 'REMOVE_TOAST', toastId: undefined })
    expect(next.toasts).toHaveLength(0)
  })

  it('returns empty array when removing from empty state', () => {
    const state = { toasts: [] }
    const next = reducer(state, { type: 'REMOVE_TOAST', toastId: undefined })
    expect(next.toasts).toHaveLength(0)
  })
})

// ---------------------------------------------------------------------------
// toast() function
// ---------------------------------------------------------------------------

describe('toast() function', () => {
  it('returns an object with id, dismiss, and update', () => {
    const result = toast({ title: 'Test' })
    expect(result.id).toBeDefined()
    expect(typeof result.id).toBe('string')
    expect(typeof result.dismiss).toBe('function')
    expect(typeof result.update).toBe('function')
  })

  it('each call returns a unique id', () => {
    const r1 = toast({ title: 'A' })
    const r2 = toast({ title: 'B' })
    expect(r1.id).not.toBe(r2.id)
  })

  it('dismiss() function can be called without throwing', () => {
    const { dismiss } = toast({ title: 'Dismiss me' })
    expect(() => dismiss()).not.toThrow()
  })

  it('update() function can be called without throwing', () => {
    const { id, update } = toast({ title: 'Original' })
    expect(() => update({ id, title: 'Updated', open: true })).not.toThrow()
  })
})

// ---------------------------------------------------------------------------
// useToast hook
// ---------------------------------------------------------------------------

describe('useToast hook', () => {
  it('returns toasts array', () => {
    const { result } = renderHook(() => useToast())
    expect(Array.isArray(result.current.toasts)).toBe(true)
  })

  it('returns toast function', () => {
    const { result } = renderHook(() => useToast())
    expect(typeof result.current.toast).toBe('function')
  })

  it('returns dismiss function', () => {
    const { result } = renderHook(() => useToast())
    expect(typeof result.current.dismiss).toBe('function')
  })

  it('calling dismiss() from hook does not throw', () => {
    const { result } = renderHook(() => useToast())
    expect(() => act(() => { result.current.dismiss() })).not.toThrow()
  })

  it('calling dismiss(id) from hook does not throw', () => {
    const { result } = renderHook(() => useToast())
    expect(() => act(() => { result.current.dismiss('nonexistent-id') })).not.toThrow()
  })
})
