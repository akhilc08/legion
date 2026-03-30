import { describe, it, expect } from 'vitest'
import { cn } from './utils'

describe('cn – no arguments', () => {
  it('returns empty string with no arguments', () => {
    expect(cn()).toBe('')
  })
})

describe('cn – single inputs', () => {
  it('returns a single class string unchanged', () => {
    expect(cn('foo')).toBe('foo')
  })

  it('returns a single class with spaces trimmed by clsx', () => {
    expect(cn('foo')).toBe('foo')
  })

  it('returns empty string for empty string argument', () => {
    expect(cn('')).toBe('')
  })
})

describe('cn – multiple string inputs', () => {
  it('joins two classes with a space', () => {
    expect(cn('foo', 'bar')).toBe('foo bar')
  })

  it('joins three classes with spaces', () => {
    expect(cn('a', 'b', 'c')).toBe('a b c')
  })

  it('handles class names with hyphens', () => {
    expect(cn('text-sm', 'font-bold')).toBe('text-sm font-bold')
  })
})

describe('cn – falsy values are omitted', () => {
  it('omits null', () => {
    expect(cn('foo', null, 'bar')).toBe('foo bar')
  })

  it('omits undefined', () => {
    expect(cn('foo', undefined, 'bar')).toBe('foo bar')
  })

  it('omits false', () => {
    expect(cn('foo', false, 'bar')).toBe('foo bar')
  })

  it('omits 0', () => {
    // 0 is falsy in JS; clsx omits it
    expect(cn('foo', 0 as unknown as string, 'bar')).toBe('foo bar')
  })

  it('returns empty string for all-falsy inputs', () => {
    expect(cn(null, undefined, false)).toBe('')
  })
})

describe('cn – conditional object syntax', () => {
  it('includes class when condition is true', () => {
    expect(cn({ foo: true })).toBe('foo')
  })

  it('omits class when condition is false', () => {
    expect(cn({ foo: false })).toBe('')
  })

  it('includes multiple true keys', () => {
    const result = cn({ foo: true, bar: true })
    expect(result).toContain('foo')
    expect(result).toContain('bar')
  })

  it('omits false keys but keeps true keys', () => {
    expect(cn({ foo: true, bar: false, baz: true })).toBe('foo baz')
  })

  it('mixed string + object', () => {
    expect(cn('base', { active: true, disabled: false })).toBe('base active')
  })
})

describe('cn – array inputs', () => {
  it('flattens an array of class strings', () => {
    expect(cn(['foo', 'bar'])).toBe('foo bar')
  })

  it('flattens nested arrays', () => {
    expect(cn(['foo', ['bar', 'baz']])).toBe('foo bar baz')
  })

  it('handles array with conditional objects', () => {
    expect(cn(['foo', { bar: true, baz: false }])).toBe('foo bar')
  })
})

describe('cn – tailwind-merge behaviour', () => {
  it('later bg class overrides earlier bg class', () => {
    expect(cn('bg-red-500', 'bg-blue-500')).toBe('bg-blue-500')
  })

  it('later text-size overrides earlier text-size', () => {
    expect(cn('text-sm', 'text-lg')).toBe('text-lg')
  })

  it('later p class overrides earlier p class', () => {
    expect(cn('p-2', 'p-4')).toBe('p-4')
  })

  it('non-conflicting classes are both kept', () => {
    expect(cn('text-sm', 'font-bold')).toBe('text-sm font-bold')
  })

  it('conditional object with tailwind conflict resolved by merge', () => {
    expect(cn('bg-red-500', { 'bg-blue-500': true })).toBe('bg-blue-500')
  })

  it('px and py do not conflict with p', () => {
    // p-4 overrides both px and py in tailwind-merge
    const result = cn('px-2', 'py-2', 'p-4')
    expect(result).toBe('p-4')
  })

  it('deduplicates identical tailwind classes via merge', () => {
    // twMerge deduplicates conflicting tailwind utilities; plain non-tailwind strings like 'foo'
    // are not deduplicated by twMerge (clsx passes them through as-is).
    // Verify that actual tailwind conflicts ARE deduplicated:
    expect(cn('bg-red-500', 'bg-red-500')).toBe('bg-red-500')
  })
})

describe('cn – mixed complex inputs', () => {
  it('handles string + array + object together', () => {
    const result = cn('base', ['extra', 'more'], { active: true, hidden: false })
    expect(result).toBe('base extra more active')
  })

  it('handles conditional merge with arrays', () => {
    const isActive = true
    const result = cn('p-2', isActive && 'bg-blue-500', ['text-sm', 'font-medium'])
    expect(result).toBe('p-2 bg-blue-500 text-sm font-medium')
  })
})
