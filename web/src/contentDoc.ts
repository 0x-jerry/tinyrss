// Builds a minimal self-contained document used as the srcdoc for content-mode
// iframes. Content is already DOMPurify-sanitized; the iframe loads it with
// sandbox="", so this embedded styling is the only styling the article receives.
export function buildContentDocument(contentHtml: string): string {
  return `<!doctype html><html><head><meta charset="utf-8">
<meta name="viewport" content="width=device-width, initial-scale=1">
<style>
  body { margin: 0; padding: 20px 28px 48px; font-size: 15px; line-height: 1.65; color: #1a1a1a; overflow-wrap: break-word; }
  img { max-width: 100%; height: auto; }
  a { color: #2563eb; }
  @media (prefers-color-scheme: dark) {
    body { background: #0f1115; color: #e5e7eb; }
    a { color: #7aa2ff; }
  }
</style>
</head><body>${contentHtml}</body></html>`
}
