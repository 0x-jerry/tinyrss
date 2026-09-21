// Builds a minimal self-contained document used as the srcdoc for content-mode
// iframes. Content is already DOMPurify-sanitized; the iframe loads it with
// sandbox="", so this embedded styling is the only styling the article receives.
// The optional header (title/meta) is injected inside the document so it scrolls
// away with the article instead of staying pinned above the iframe.

export interface ContentDocHeader {
  title: string
  meta: string
}

function escapeHtml(s: string): string {
  return s
    .replace(/&/g, '&amp;')
    .replace(/</g, '&lt;')
    .replace(/>/g, '&gt;')
    .replace(/"/g, '&quot;')
    .replace(/'/g, '&#39;')
}

export function buildContentDocument(contentHtml: string, header?: ContentDocHeader): string {
  const head = header
    ? `<div class="article-head"><h1 class="article-title">${escapeHtml(header.title)}</h1><div class="article-meta">${escapeHtml(header.meta)}</div></div>`
    : ''
  return `<!doctype html><html><head><meta charset="utf-8">
<meta name="viewport" content="width=device-width, initial-scale=1">
<style>
  body { margin: 0; padding: 20px 28px 48px; font-size: 15px; line-height: 1.65; color: #1a1a1a; overflow-wrap: break-word; }
  .article-head { margin-bottom: 18px; padding-bottom: 12px; border-bottom: 1px solid rgba(127,127,127,.3); }
  .article-title { margin: 0; font-size: 24px; font-weight: 600; line-height: 1.3; }
  .article-meta { margin-top: 8px; font-size: 13px; opacity: .6; }
  img { max-width: 100%; height: auto; }
  a { color: #2563eb; }
  @media (prefers-color-scheme: dark) {
    body { background: #0f1115; color: #e5e7eb; }
    a { color: #7aa2ff; }
    .article-head { border-bottom-color: rgba(255,255,255,.18); }
  }
</style>
</head><body>${head}${contentHtml}</body></html>`
}
