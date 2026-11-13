package feeds

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
)

// maxFaviconSize caps the bytes read from the favicon service so a hostile or
// runaway upstream response cannot exhaust memory.
const maxFaviconSize = 256 << 10

// faviconBaseURL is the upstream favicon service. It is a package variable so
// tests can point it at a local server; production always uses Google's S2
// favicon service.
var faviconBaseURL = "https://www.google.com/s2/favicons"

// FetchFavicon fetches a site icon for host from Google's favicon service and
// returns its content type and bytes. The caller is responsible for validating
// host (see the HTTP handler); this method only builds the upstream query, so
// the dialed host is always the fixed service, never the caller-supplied host.
func (f *Fetcher) FetchFavicon(ctx context.Context, host string, size int) (contentType string, body []byte, err error) {
	u := faviconBaseURL + "?domain=" + url.QueryEscape(host) + "&sz=" + strconv.Itoa(size)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return "", nil, err
	}
	req.Header.Set("User-Agent", userAgent)
	resp, err := f.client.Do(req)
	if err != nil {
		return "", nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", nil, fmt.Errorf("favicon upstream returned %s", resp.Status)
	}
	body, err = io.ReadAll(io.LimitReader(resp.Body, maxFaviconSize))
	if err != nil {
		return "", nil, err
	}
	return resp.Header.Get("Content-Type"), body, nil
}

// SetFaviconBaseURL overrides the favicon service base URL and returns a func
// that restores the previous value. Intended for tests in other packages (e.g.
// the HTTP handler tests) that need to stub the upstream; production never
// calls it.
func SetFaviconBaseURL(baseURL string) (restore func()) {
	prev := faviconBaseURL
	faviconBaseURL = baseURL
	return func() { faviconBaseURL = prev }
}
