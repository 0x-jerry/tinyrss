package feeds

import (
	"bytes"
	"fmt"
	"html"
	"io"
	"net/http"
	"net/url"
	"strings"

	readability "codeberg.org/readeck/go-readability/v2"
	nethtml "golang.org/x/net/html"
)

const renderBodyLimit = 8 << 20 // cap proxied article bodies

// FetchRender downloads an article URL server-side, extracts its meaningful
// content, strips scripts, wraps it in a reader-styled document, and caches the
// result in the render_cache table keyed by URL. Only http/https are allowed;
// non-200 responses are errors. Cached results are returned without re-fetching.
func (f *Fetcher) FetchRender(rawURL, proxyURL string) ([]byte, error) {
	u, err := url.Parse(rawURL)
	if err != nil {
		return nil, err
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return nil, fmt.Errorf("unsupported scheme %q", u.Scheme)
	}
	key := u.String()
	if cached, ok, _ := f.repo.GetRenderCache(key); ok {
		return cached, nil
	}
	f.renderSem <- struct{}{}
	defer func() { <-f.renderSem }()
	v, err, _ := f.renderSf.Do(key, func() (any, error) {
		if cached, ok, _ := f.repo.GetRenderCache(key); ok {
			return cached, nil
		}
		body, err := f.fetchPage(u, proxyURL)
		if err != nil {
			return nil, err
		}
		title, inner := renderArticle(body, u)
		out := wrapRender(title, key, inner)
		// A failed write only loses this article's cache entry, not the response.
		_ = f.repo.PutRenderCache(key, out)
		return out, nil
	})
	if err != nil {
		return nil, err
	}
	return v.([]byte), nil
}

func (f *Fetcher) fetchPage(u *url.URL, proxyURL string) ([]byte, error) {
	client, err := f.renderClientFor(proxyURL)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(f.ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", userAgent)
	req.Header.Set("Accept", "text/html")
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("upstream returned %s", resp.Status)
	}
	return io.ReadAll(io.LimitReader(resp.Body, renderBodyLimit))
}

// renderArticle extracts the meaningful content of an HTML page. It prefers a
// readability extraction of the main article; when that yields nothing the raw
// page body is used. Either way the result is script-stripped.
func renderArticle(body []byte, u *url.URL) (title string, inner []byte) {
	if article, err := readability.FromReader(bytes.NewReader(body), u); err == nil && article.Node != nil {
		sanitize(article.Node)
		if out := renderNode(article.Node); !isBlank(out) {
			return article.Title(), out
		}
	}
	doc, err := nethtml.Parse(bytes.NewReader(body))
	if err != nil {
		return "", []byte("<p>Could not extract article content.</p>")
	}
	sanitize(doc)
	title = findTitle(doc)
	if bodyNode := findBody(doc); bodyNode != nil {
		return title, renderChildren(bodyNode)
	}
	return title, renderChildren(doc)
}

// stripTags are dropped wholesale; they can never be read and may execute code.
var stripTags = map[string]bool{
	"script": true, "noscript": true, "style": true,
	"iframe": true, "object": true, "embed": true,
}

// sanitize walks a node tree, removes stripTags and neutralises inline event
// handlers and script URLs, in place.
func sanitize(n *nethtml.Node) {
	for c := n.FirstChild; c != nil; {
		next := c.NextSibling
		if c.Type == nethtml.ElementNode && stripTags[strings.ToLower(c.Data)] {
			n.RemoveChild(c)
		} else {
			sanitize(c)
		}
		c = next
	}
	if n.Type != nethtml.ElementNode {
		return
	}
	attrs := n.Attr[:0]
	for _, a := range n.Attr {
		if isEventHandler(a) || isDangerousURL(a) {
			continue
		}
		attrs = append(attrs, a)
	}
	n.Attr = attrs
}

func isEventHandler(a nethtml.Attribute) bool { return strings.HasPrefix(strings.ToLower(a.Key), "on") }

// urlAttrs carry a URL that could be a script vector.
var urlAttrs = map[string]bool{
	"href": true, "src": true, "xlink:href": true,
	"poster": true, "action": true, "formaction": true, "data": true,
}

func isDangerousURL(a nethtml.Attribute) bool {
	if !urlAttrs[strings.ToLower(a.Key)] {
		return false
	}
	scheme := strings.ToLower(strings.TrimSpace(a.Val))
	if i := strings.IndexByte(scheme, ':'); i >= 0 {
		scheme = scheme[:i]
	}
	return scheme == "javascript" || scheme == "vbscript" || scheme == "data"
}

func isBlank(b []byte) bool {
	for _, r := range b {
		if !strings.ContainsRune(" \t\r\n", rune(r)) {
			return false
		}
	}
	return true
}

func renderNode(n *nethtml.Node) []byte {
	var buf bytes.Buffer
	_ = nethtml.Render(&buf, n)
	return buf.Bytes()
}

func renderChildren(n *nethtml.Node) []byte {
	var buf bytes.Buffer
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		_ = nethtml.Render(&buf, c)
	}
	return buf.Bytes()
}

func findTitle(n *nethtml.Node) string {
	if n.Type == nethtml.ElementNode && strings.ToLower(n.Data) == "title" {
		return strings.TrimSpace(string(renderChildren(n)))
	}
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		if t := findTitle(c); t != "" {
			return t
		}
	}
	return ""
}

func findBody(n *nethtml.Node) *nethtml.Node {
	if n.Type == nethtml.ElementNode && strings.ToLower(n.Data) == "body" {
		return n
	}
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		if b := findBody(c); b != nil {
			return b
		}
	}
	return nil
}

// wrapRender embeds extracted content in a self-contained document with the
// app's reader styling. The <base> lets relative URLs resolve against the
// original page in the srcdoc iframe.
func wrapRender(title, baseURL string, inner []byte) []byte {
	var b bytes.Buffer
	b.WriteString("<!doctype html><html lang=\"en\"><head><meta charset=\"utf-8\">")
	b.WriteString(`<meta name="viewport" content="width=device-width, initial-scale=1">`)

	b.WriteString(`<base href="`)
	b.WriteString(html.EscapeString(baseURL))
	b.WriteString(`">`)

	b.WriteString(`<title>`)
	b.WriteString(html.EscapeString(title))
	b.WriteString(`</title>`)

	b.WriteString(`<style>` + renderCSS + `</style></head><body><main class="tinyrss-article">`)
	b.Write(inner)
	b.WriteString(`</main></body></html>`)
	return b.Bytes()
}

const renderCSS = `
:root { color-scheme: light dark; --bg:#ffffff; --text:#1f2328; --muted:#6a737d; --link:#0969da; --border:#d1d9e0; --code:#f6f8fa; }
@media (prefers-color-scheme: dark) {
  :root { --bg:#0d1117; --text:#e6edf3; --muted:#8b949e; --link:#58a6ff; --border:#30363d; --code:#161b22; }
}
* { box-sizing: border-box; }
body { margin: 0 auto; padding: 28px 24px 64px; max-width: 760px; font: 16px/1.7 -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, Helvetica, Arial, sans-serif; color: var(--text); background: var(--bg); overflow-wrap: break-word; }
h1, h2, h3, h4 { line-height: 1.3; margin: 1.4em 0 0.6em; }
h1 { font-size: 1.7em; } h2 { font-size: 1.4em; } h3 { font-size: 1.2em; } h4 { font-size: 1.05em; }
p { margin: 0 0 1em; }
a { color: var(--link); text-decoration: none; } a:hover { text-decoration: underline; }
img, video { max-width: 100%; height: auto; border-radius: 6px; }
figure { margin: 1em 0; text-align: center; } figcaption { margin-top: 6px; font-size: 13px; color: var(--muted); }
pre { overflow-x: auto; padding: 12px 14px; border-radius: 8px; background: var(--code); font-size: 14px; line-height: 1.5; }
code { font-family: ui-monospace, SFMono-Regular, "SF Mono", Menlo, Consolas, monospace; font-size: 0.9em; background: var(--code); padding: 0.15em 0.4em; border-radius: 5px; }
pre code { background: none; padding: 0; }
blockquote { margin: 1em 0; padding: 0 1em; border-left: 3px solid var(--border); color: var(--muted); }
hr { border: 0; border-top: 1px solid var(--border); margin: 1.6em 0; }
table { border-collapse: collapse; width: 100%; margin: 1em 0; font-size: 14px; }
th, td { border: 1px solid var(--border); padding: 8px 10px; text-align: left; }
th { background: var(--code); }
ul, ol { padding-left: 1.4em; }
.tinyrss-article > :first-child { margin-top: 0; }
`
