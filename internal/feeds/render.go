package feeds

import (
	"bytes"
	"fmt"
	"html"
	"io"
	"net/http"
	"net/url"
)

const renderBodyLimit = 8 << 20 // cap proxied article bodies

// FetchRender downloads an article URL server-side and returns its HTML with a
// <base> injected so relative links/images resolve against the original site.
// Only http/https are allowed; non-200 responses are errors.
func (f *Fetcher) FetchRender(rawURL string) ([]byte, error) {
	u, err := url.Parse(rawURL)
	if err != nil {
		return nil, err
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return nil, fmt.Errorf("unsupported scheme %q", u.Scheme)
	}
	req, err := http.NewRequest(http.MethodGet, u.String(), nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", userAgent)
	req.Header.Set("Accept", "text/html")
	resp, err := f.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("upstream returned %s", resp.Status)
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, renderBodyLimit))
	if err != nil {
		return nil, err
	}
	return injectBase(body, u.String()), nil
}

func injectBase(body []byte, baseURL string) []byte {
	tag := []byte(`<base href="` + html.EscapeString(baseURL) + `">`)
	lower := bytes.ToLower(body)
	head := bytes.Index(lower, []byte("<head"))
	if head < 0 {
		return append(tag, body...)
	}
	// Insert after the closing '>' of the <head ...> opening tag.
	end := bytes.IndexByte(body[head:], '>')
	if end < 0 {
		return append(tag, body...)
	}
	out := make([]byte, 0, len(body)+len(tag))
	out = append(out, body[:head+end+1]...)
	out = append(out, tag...)
	out = append(out, body[head+end+1:]...)
	return out
}
