package feeds

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"

	"golang.org/x/net/proxy"
)

const httpTimeout = 20 * time.Second

func baseTransport() *http.Transport {
	return &http.Transport{
		MaxIdleConns:        workerCount * 2,
		MaxIdleConnsPerHost: workerCount,
		IdleConnTimeout:     90 * time.Second,
		ForceAttemptHTTP2:   true,
	}
}

func newHTTPClient(rt http.RoundTripper) *http.Client {
	return &http.Client{
		Timeout:   httpTimeout,
		Transport: rt,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if len(via) >= 5 {
				return errors.New("too many redirects")
			}
			return nil
		},
	}
}

// ValidateProxyURL rejects a proxy that cannot be dialed by scheme; empty means
// a direct connection. It does not probe connectivity.
func ValidateProxyURL(raw string) error {
	if strings.TrimSpace(raw) == "" {
		return nil
	}
	_, err := parseProxy(raw)
	return err
}

func parseProxy(raw string) (*url.URL, error) {
	u, err := url.Parse(strings.TrimSpace(raw))
	if err != nil {
		return nil, fmt.Errorf("invalid proxy url: %w", err)
	}
	switch strings.ToLower(u.Scheme) {
	case "http", "https", "socks5", "socks5h":
	default:
		return nil, fmt.Errorf("unsupported proxy scheme %q", u.Scheme)
	}
	if u.Host == "" {
		return nil, errors.New("proxy url must include a host")
	}
	return u, nil
}

// clientFor returns the HTTP client for a feed's proxy, building and caching
// one per distinct proxy URL. An empty proxy yields the shared direct client.
func (f *Fetcher) clientFor(proxyURL string) (*http.Client, error) {
	proxyURL = strings.TrimSpace(proxyURL)
	if proxyURL == "" {
		return f.client, nil
	}
	f.proxyMu.Lock()
	defer f.proxyMu.Unlock()
	if c, ok := f.proxied[proxyURL]; ok {
		return c, nil
	}
	c, err := newProxiedClient(proxyURL)
	if err != nil {
		return nil, err
	}
	f.proxied[proxyURL] = c
	return c, nil
}

func newProxiedClient(raw string) (*http.Client, error) {
	u, err := parseProxy(raw)
	if err != nil {
		return nil, err
	}
	tr := baseTransport()
	switch strings.ToLower(u.Scheme) {
	case "http", "https":
		tr.Proxy = http.ProxyURL(u)
	case "socks5", "socks5h":
		dialer, err := socksDialer(u)
		if err != nil {
			return nil, err
		}
		tr.DialContext = dialContext(dialer)
	}
	return newHTTPClient(tr), nil
}

func socksDialer(u *url.URL) (proxy.Dialer, error) {
	var auth *proxy.Auth
	if u.User != nil {
		auth = &proxy.Auth{User: u.User.Username()}
		if pass, ok := u.User.Password(); ok {
			auth.Password = pass
		}
	}
	return proxy.SOCKS5("tcp", u.Host, auth, proxy.Direct)
}

func dialContext(d proxy.Dialer) func(context.Context, string, string) (net.Conn, error) {
	if cd, ok := d.(proxy.ContextDialer); ok {
		return cd.DialContext
	}
	return func(_ context.Context, network, addr string) (net.Conn, error) {
		return d.Dial(network, addr)
	}
}
