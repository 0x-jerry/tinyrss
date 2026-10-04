import { describe, it, expect } from 'vitest'
import { externalUrl } from '../src/helpers/externalUrl'

const base = 'http://localhost/'

describe('externalUrl', () => {
  it('keeps http(s) urls', () => {
    expect(externalUrl('https://example.com/a?b=1', base)).toBe('https://example.com/a?b=1')
    expect(externalUrl('http://example.com/', base)).toBe('http://example.com/')
  })

  it('resolves relative urls against the base', () => {
    expect(externalUrl('/post/1', 'https://example.com/feed')).toBe('https://example.com/post/1')
  })

  it('rejects every other scheme, including script and data urls', () => {
    expect(externalUrl('javascript:alert(1)', base)).toBeNull()
    expect(externalUrl('data:text/html,<script>alert(1)</script>', base)).toBeNull()
    expect(externalUrl('mailto:a@b.c', base)).toBeNull()
  })

  it('rejects empty and unparseable values', () => {
    expect(externalUrl(null, base)).toBeNull()
    expect(externalUrl(undefined, base)).toBeNull()
    expect(externalUrl('', base)).toBeNull()
    expect(externalUrl('http://[', base)).toBeNull()
  })
})
