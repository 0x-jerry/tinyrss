import { describe, it, expect } from 'vitest'
import { sanitizeTheme } from '../src/providers/theme'

describe('sanitizeTheme', () => {
  it('returns system for empty or undefined input', () => {
    expect(sanitizeTheme(undefined)).toBe('system')
    expect(sanitizeTheme(null)).toBe('system')
    expect(sanitizeTheme('')).toBe('system')
  })

  it('keeps valid modes', () => {
    expect(sanitizeTheme('light')).toBe('light')
    expect(sanitizeTheme('dark')).toBe('dark')
    expect(sanitizeTheme('system')).toBe('system')
  })

  it('coerces invalid or legacy values to system', () => {
    expect(sanitizeTheme('blue')).toBe('system')
    expect(sanitizeTheme(123)).toBe('system')
    expect(sanitizeTheme({ mode: 'dark' })).toBe('system')
  })
})
