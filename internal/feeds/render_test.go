package feeds

import (
	"bytes"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

const articlePage = `<!doctype html><html><head><title>Sample Article</title>
<style>.junk{color:red}</style>
<script>window.evil = 1</script>
</head><body>
<nav><a href="/">Home</a></nav>
<script>alert('x')</script>
<article>
<h1>Sample Article</h1>
<p onclick="evil()">Hello meaningful content here.</p>
<p>Second paragraph of the story.</p>
<img src="/pic.jpg" onerror="alert(1)">
</article>
<footer>footer junk</footer>
</body></html>`

// servePage returns an httptest server serving html on every path, counting hits.
func servePage(t *testing.T, html string) (*httptest.Server, *int64) {
	t.Helper()
	var hits int64
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt64(&hits, 1)
		_, _ = w.Write([]byte(html))
	}))
	t.Cleanup(srv.Close)
	return srv, &hits
}

func TestFetchRenderExtractsAndStrips(t *testing.T) {
	repo := newTestRepo(t)
	f := NewFetcher(repo)
	srv, hits := servePage(t, articlePage)

	out, err := f.FetchRender(srv.URL + "/art", "")
	if err != nil {
		t.Fatal(err)
	}
	body := string(out)

	if !bytes.HasPrefix(bytes.TrimSpace(out), []byte("<!doctype html>")) {
		t.Fatalf("missing doctype: %s", body)
	}
	if !strings.Contains(body, `class="tinyrss-article"`) {
		t.Fatalf("missing styled wrapper: %s", body)
	}
	if !strings.Contains(body, `<base href="`+srv.URL+`/art">`) {
		t.Fatalf("missing base href: %s", body)
	}
	if !strings.Contains(body, "Hello meaningful content here.") {
		t.Fatalf("extracted content missing: %s", body)
	}
	if !strings.Contains(body, "Sample Article") {
		t.Fatalf("title missing: %s", body)
	}
	for _, bad := range []string{"<script", "onclick", "onerror", ".junk"} {
		if strings.Contains(body, bad) {
			t.Fatalf("sanitize failed, found %q: %s", bad, body)
		}
	}

	// Same URL hits the DB cache, so the upstream is fetched only once.
	if _, err := f.FetchRender(srv.URL + "/art", ""); err != nil {
		t.Fatal(err)
	}
	if n := atomic.LoadInt64(hits); n != 1 {
		t.Fatalf("upstream hits = %d, want 1 (cached)", n)
	}
}

func TestFetchRenderFallbackForNonArticle(t *testing.T) {
	repo := newTestRepo(t)
	f := NewFetcher(repo)
	srv, _ := servePage(t, `<html><body><p>just a bare page</p><script>evil()</script></body></html>`)

	out, err := f.FetchRender(srv.URL + "/", "")
	if err != nil {
		t.Fatal(err)
	}
	body := string(out)
	if !strings.Contains(body, "just a bare page") {
		t.Fatalf("fallback content missing: %s", body)
	}
	if !strings.Contains(body, `class="tinyrss-article"`) {
		t.Fatalf("fallback missing styled wrapper: %s", body)
	}
	if strings.Contains(body, "<script") {
		t.Fatalf("fallback left a script tag: %s", body)
	}
}

// TestRenderConcurrentSameURLCoalesces pins that simultaneous requests for one
// article share a single upstream fetch/render.
func TestRenderConcurrentSameURLCoalesces(t *testing.T) {
	repo := newTestRepo(t)
	f := NewFetcher(repo)

	var hits int64
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt64(&hits, 1)
		time.Sleep(50 * time.Millisecond)
		_, _ = w.Write([]byte(articlePage))
	}))
	t.Cleanup(srv.Close)
	url := srv.URL + "/art"

	const callers = 8
	start := make(chan struct{})
	results := make([][]byte, callers)
	errs := make([]error, callers)
	var wg sync.WaitGroup
	for i := 0; i < callers; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			<-start
			results[i], errs[i] = f.FetchRender(url, "")
		}(i)
	}
	close(start)
	wg.Wait()

	if n := atomic.LoadInt64(&hits); n != 1 {
		t.Fatalf("upstream hits = %d, want 1 (coalesced)", n)
	}
	for i := range callers {
		if errs[i] != nil {
			t.Fatalf("caller %d: %v", i, errs[i])
		}
		if !bytes.Equal(results[i], results[0]) {
			t.Fatalf("caller %d got different bytes", i)
		}
	}
}

// TestRenderConcurrencyBounded pins that distinct article fetches never exceed
// renderWorkers in flight, so DOM trees and bodies stay bounded.
func TestRenderConcurrencyBounded(t *testing.T) {
	repo := newTestRepo(t)
	f := NewFetcher(repo)

	var inFlight, maxInFlight int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cur := atomic.AddInt32(&inFlight, 1)
		for {
			max := atomic.LoadInt32(&maxInFlight)
			if cur <= max || atomic.CompareAndSwapInt32(&maxInFlight, max, cur) {
				break
			}
		}
		time.Sleep(30 * time.Millisecond)
		atomic.AddInt32(&inFlight, -1)
		_, _ = w.Write([]byte(articlePage))
	}))
	t.Cleanup(srv.Close)

	const callers = 10
	var wg sync.WaitGroup
	for i := 0; i < callers; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			_, _ = f.FetchRender(fmt.Sprintf("%s/art-%d", srv.URL, i), "")
		}(i)
	}
	wg.Wait()

	if got := atomic.LoadInt32(&maxInFlight); got > renderWorkers {
		t.Fatalf("concurrent renders = %d, want <= %d", got, renderWorkers)
	}
}
