import { describe, it, expect } from 'vitest'
import { QueryClient } from '@tanstack/react-query'
import { queryClient } from './queryClient'

describe('queryClient', () => {
  it('is defined', () => {
    expect(queryClient).toBeDefined()
  })

  it('is a QueryClient instance', () => {
    expect(queryClient).toBeInstanceOf(QueryClient)
  })

  it('has invalidateQueries method', () => {
    expect(typeof queryClient.invalidateQueries).toBe('function')
  })

  it('has getQueryData method', () => {
    expect(typeof queryClient.getQueryData).toBe('function')
  })

  it('has setQueryData method', () => {
    expect(typeof queryClient.setQueryData).toBe('function')
  })

  it('has default staleTime of 5 minutes', () => {
    const defaults = queryClient.getDefaultOptions()
    expect(defaults.queries?.staleTime).toBe(5 * 60 * 1000)
  })

  it('has retry set to 1', () => {
    const defaults = queryClient.getDefaultOptions()
    expect(defaults.queries?.retry).toBe(1)
  })
})
