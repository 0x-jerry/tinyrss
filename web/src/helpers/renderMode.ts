// Render modes for a feed. Mirrors the backend feeds.render_mode column.
export const RenderMode = { Content: 0, Iframe: 1, Server: 2 } as const

export type RenderKind = 'content' | 'iframe' | 'server'

// renderKind decides how an article is shown for a feed's mode. Server/iframe
// modes need a URL; without one (or in content mode) we show the parsed body.
export function renderKind(mode: number, url: string | null): RenderKind {
  if (!url) return 'content'
  if (mode === RenderMode.Iframe) return 'iframe'
  if (mode === RenderMode.Server) return 'server'
  return 'content'
}
