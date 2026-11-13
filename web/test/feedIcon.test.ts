import { describe, it, expect } from 'vitest'
import { feedHost, feedIconUrl, clampIconSize } from '../src/helpers/feedIcon'

describe('feedHost', () => {
  it('uses the site_url host', () => {
    expect(feedHost({ site_url: 'https://site.example/post/1', feed_url: 'https://feed.example/rss' })).toBe('site.example')
  })

  it('falls back to the feed_url host when site_url is empty', () => {
    expect(feedHost({ site_url: '', feed_url: 'https://feed.example/rss' })).toBe('feed.example')
  })

  it('returns null for missing or non-http(s) urls', () => {
    expect(feedHost(null)).toBeNull()
    expect(feedHost({})).toBeNull()
    expect(feedHost({ site_url: '', feed_url: '' })).toBeNull()
    expect(feedHost({ site_url: 'javascript:alert(1)' })).toBeNull()
    expect(feedHost({ site_url: 'data:x', feed_url: 'ftp://a.b/c' })).toBeNull()
  })

  it('ignores a malformed site_url and tries feed_url', () => {
    expect(feedHost({ site_url: 'http://[', feed_url: 'https://ok.example/rss' })).toBe('ok.example')
  })
})

describe('clampIconSize', () => {
  it('snaps to the nearest supported bucket', () => {
    expect(clampIconSize(14)).toBe(16)
    expect(clampIconSize(999)).toBe(256)
    expect(clampIconSize(40)).toBe(32)
    expect(clampIconSize(100)).toBe(128)
    expect(clampIconSize(NaN)).toBe(32)
  })
})

describe('feedIconUrl', () => {
  it('builds the same-origin proxy url with the encoded host', () => {
    expect(feedIconUrl({ site_url: 'https://site.example/post/1' })).toBe('/api/favicon?domain=site.example&sz=32')
  })

  it('clamps the size parameter', () => {
    expect(feedIconUrl({ site_url: 'https://site.example/' }, 999)).toBe('/api/favicon?domain=site.example&sz=256')
  })

  it('returns null when no host can be derived', () => {
    expect(feedIconUrl(null)).toBeNull()
    expect(feedIconUrl({ site_url: '', feed_url: '' })).toBeNull()
  })
})
