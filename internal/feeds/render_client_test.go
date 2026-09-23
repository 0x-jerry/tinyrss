package feeds

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
)

func TestFetchRenderImpersonatesChrome(t *testing.T) {
	f := NewFetcher(newTestRepo(t))

	var mu sync.Mutex
	var ua string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		ua = r.Header.Get("User-Agent")
		mu.Unlock()
		_, _ = w.Write([]byte(articlePage))
	}))
	t.Cleanup(srv.Close)

	if _, err := f.FetchRender(srv.URL+"/art", ""); err != nil {
		t.Fatal(err)
	}
	mu.Lock()
	defer mu.Unlock()
	if !strings.Contains(ua, "Chrome/") {
		t.Fatalf("User-Agent = %q, want Chrome impersonation", ua)
	}
}

func TestRenderClientForCachesByProxy(t *testing.T) {
	f := NewFetcher(newTestRepo(t))
	const proxy = "socks5://user:pass@127.0.0.1:1080"

	c, err := f.renderClientFor(proxy)
	if err != nil {
		t.Fatalf("renderClientFor: %v", err)
	}
	if c == nil {
		t.Fatal("renderClientFor returned nil client")
	}
	if again, _ := f.renderClientFor(proxy); again != c {
		t.Fatal("renderClientFor did not cache per proxy")
	}
	if _, err := f.renderClientFor("ftp://127.0.0.1:1080"); err == nil {
		t.Fatal("renderClientFor accepted an unsupported proxy scheme")
	}
}
