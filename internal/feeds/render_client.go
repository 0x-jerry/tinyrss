package feeds

import (
	"net/http"
	"strings"

	"github.com/enetx/g"
	"github.com/enetx/surf"
)

// newRenderClient builds a browser-impersonating client for article rendering,
// so pages that reject a plain client still render.
func newRenderClient(proxyURL string) (*http.Client, error) {
	proxyURL = strings.TrimSpace(proxyURL)
	if proxyURL != "" {
		if _, err := parseProxy(proxyURL); err != nil {
			return nil, err
		}
	}
	b := surf.NewClient().Builder().
		Impersonate().Chrome().
		Timeout(httpTimeout).
		MaxRedirects(5).
		Proxy(g.String(proxyURL)) // empty clears the transport's env-proxy default
	c, err := b.Build().Result()
	if err != nil {
		return nil, err
	}
	return c.Std(), nil
}

func (f *Fetcher) renderClientFor(proxyURL string) (*http.Client, error) {
	proxyURL = strings.TrimSpace(proxyURL)
	return f.renderPool.get(proxyURL, func() (*http.Client, error) {
		return newRenderClient(proxyURL)
	})
}
