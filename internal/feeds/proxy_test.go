package feeds

import (
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"

	"tinyrss/internal/repository"
)

func TestValidateProxyURL(t *testing.T) {
	for _, raw := range []string{
		"",
		"http://127.0.0.1:3128",
		"https://proxy.example:8443",
		"socks5://user:pass@127.0.0.1:1080",
		"socks5h://host:1080",
	} {
		if err := ValidateProxyURL(raw); err != nil {
			t.Errorf("ValidateProxyURL(%q) = %v, want nil", raw, err)
		}
	}
	for _, raw := range []string{
		"127.0.0.1:1080",
		"ftp://host:21",
		"http://",
		"://bad",
	} {
		if err := ValidateProxyURL(raw); err == nil {
			t.Errorf("ValidateProxyURL(%q) = nil, want error", raw)
		}
	}
}

func TestClientForBuildsAndCachesSocks5(t *testing.T) {
	f := NewFetcher(newTestRepo(t))
	c, err := f.clientFor("socks5://user:pass@127.0.0.1:1080")
	if err != nil {
		t.Fatalf("clientFor socks5: %v", err)
	}
	if c == nil {
		t.Fatal("clientFor returned a nil client")
	}
	if again, _ := f.clientFor("socks5://user:pass@127.0.0.1:1080"); again != c {
		t.Fatal("expected the same cached client for one proxy URL")
	}
}

// TestRefreshUsesFeedProxy points the feed at an unresolvable host and lets an
// HTTP proxy answer instead: success proves the fetch went through the proxy.
func TestRefreshUsesFeedProxy(t *testing.T) {
	var hits int32
	proxySrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&hits, 1)
		_, _ = w.Write([]byte(rssXML))
	}))
	defer proxySrv.Close()

	repo := newTestRepo(t)
	feed, err := repo.CreateFeed(repository.Feed{
		Title:    "F",
		FeedURL:  "http://feed.invalid/rss",
		ProxyURL: proxySrv.URL,
	})
	if err != nil {
		t.Fatal(err)
	}
	fetcher := NewFetcher(repo)
	newCount, err := fetcher.RefreshFeed(feed.ID)
	if err != nil {
		t.Fatalf("refresh through proxy: %v", err)
	}
	if newCount != 1 {
		t.Fatalf("newCount = %d, want 1", newCount)
	}
	if atomic.LoadInt32(&hits) == 0 {
		t.Fatal("proxy server was never hit")
	}
}
