// A feed's site icon is resolved from its site_url (preferred) or feed_url and
// loaded through TinyRSS's own /api/favicon proxy, which fronts Google's
// favicon service. Deriving the host here keeps the browser same-origin.

export interface FeedIconSource {
  site_url?: string
  feed_url?: string
}

const SIZES = [16, 32, 64, 128, 256] as const

// feedHost returns the site host to look up an icon for: site_url's host when
// it is a usable http(s) URL, else feed_url's host, else null. Only http(s) URLs
// are honored so a stray javascript:/data: value can never be used.
export function feedHost(feed: FeedIconSource | null | undefined): string | null {
  if (!feed) return null
  for (const candidate of [feed.site_url, feed.feed_url]) {
    if (!candidate) continue
    try {
      const u = new URL(candidate)
      if ((u.protocol === 'http:' || u.protocol === 'https:') && u.hostname) {
        return u.hostname
      }
    } catch {
      // ignore malformed URL and try the next candidate
    }
  }
  return null
}

// clampIconSize snaps a requested size to the nearest bucket the favicon service
// supports.
export function clampIconSize(size: number): number {
  if (!Number.isFinite(size)) return 32
  let best: number = SIZES[0]
  for (const s of SIZES) {
    if (Math.abs(s - size) < Math.abs(best - size)) best = s
  }
  return best
}

// feedIconUrl builds the same-origin proxy URL for a feed's icon, or null when
// no host can be derived (the caller should fall back to a glyph).
export function feedIconUrl(feed: FeedIconSource | null | undefined, size = 32): string | null {
  const host = feedHost(feed)
  if (!host) return null
  return `/api/favicon?domain=${encodeURIComponent(host)}&sz=${clampIconSize(size)}`
}
