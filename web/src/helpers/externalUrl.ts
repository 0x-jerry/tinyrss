// Only http(s) article URLs may be used as an href, so a `javascript:`/`data:`
// url stored with an item can never become a clickable link in the reader.
export function externalUrl(url: string | null | undefined, base?: string): string | null {
  if (!url) return null
  try {
    const parsed = new URL(url, base)
    return parsed.protocol === 'http:' || parsed.protocol === 'https:' ? parsed.href : null
  } catch {
    return null
  }
}
