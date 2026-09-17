package server

import (
	"bytes"
	"encoding/json"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"strings"
	"testing"
	"testing/fstest"

	"tinyrss/internal/feeds"
	"tinyrss/internal/store"
)

const token = "secret-token"

const testRSS = `<?xml version="1.0" encoding="UTF-8"?>
<rss version="2.0"><channel>
<title>Example Blog</title>
<link>https://example.com/</link>
<description>desc</description>
<item><title>Hello</title><link>https://example.com/1</link><guid>g1</guid>
<pubDate>Mon, 02 Jan 2006 15:04:05 +0000</pubDate><description>sum</description></item>
</channel></rss>`

func newTestServer(t *testing.T) (*httptest.Server, *feeds.Repo) {
	t.Helper()
	st, err := store.Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	repo := feeds.NewRepo(st.DB)
	fetcher := feeds.NewFetcher(repo)
	spa := fstest.MapFS{"index.html": &fstest.MapFile{Data: []byte("<h1>tinyrss</h1>")}}
	s := New(repo, fetcher, token, spa)
	ts := httptest.NewServer(s.Handler())
	t.Cleanup(func() {
		ts.Close()
		st.Close()
	})
	return ts, repo
}

func do(t *testing.T, ts *httptest.Server, method, path, token string, body io.Reader) (*http.Response, []byte) {
	t.Helper()
	req, err := http.NewRequest(method, ts.URL+path, body)
	if err != nil {
		t.Fatal(err)
	}
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	data, _ := io.ReadAll(resp.Body)
	return resp, data
}

func decode[T any](t *testing.T, data []byte) T {
	t.Helper()
	var v T
	if err := json.Unmarshal(data, &v); err != nil {
		t.Fatalf("decode %s: %v", data, err)
	}
	return v
}

func TestAuthGuard(t *testing.T) {
	ts, _ := newTestServer(t)

	// Health is auth-exempt.
	resp, _ := do(t, ts, "GET", "/api/health", "", nil)
	if resp.StatusCode != 200 {
		t.Fatalf("health status = %d", resp.StatusCode)
	}

	// Everything else requires a token.
	resp, body := do(t, ts, "GET", "/api/feeds", "", nil)
	if resp.StatusCode != 401 {
		t.Fatalf("no-token status = %d, body=%s", resp.StatusCode, body)
	}
	if decode[map[string]string](t, body)["error"] != "unauthorized" {
		t.Fatalf("unexpected body %s", body)
	}

	// Wrong token is rejected, right token passes.
	if resp, _ := do(t, ts, "GET", "/api/feeds", "wrong", nil); resp.StatusCode != 401 {
		t.Fatalf("wrong-token status = %d", resp.StatusCode)
	}
	if resp, _ := do(t, ts, "GET", "/api/feeds", token, nil); resp.StatusCode != 200 {
		t.Fatalf("valid-token status = %d", resp.StatusCode)
	}
}

func TestFeedLifecycleAndItems(t *testing.T) {
	ts, _ := newTestServer(t)
	feedSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("If-None-Match") == `"e1"` {
			w.WriteHeader(http.StatusNotModified)
			return
		}
		w.Header().Set("ETag", `"e1"`)
		_, _ = w.Write([]byte(testRSS))
	}))
	defer feedSrv.Close()

	// Create folder + feed.
	resp, body := do(t, ts, "POST", "/api/folders", token, strings.NewReader(`{"name":"Tech"}`))
	if resp.StatusCode != 200 {
		t.Fatalf("create folder: %d %s", resp.StatusCode, body)
	}
	folder := decode[feeds.Folder](t, body)
	if folder.ID == 0 || folder.Name != "Tech" {
		t.Fatalf("unexpected folder %+v", folder)
	}

	resp, body = do(t, ts, "POST", "/api/feeds", token, strings.NewReader(`{"feed_url":"`+feedSrv.URL+`/feed"}`))
	if resp.StatusCode != 200 {
		t.Fatalf("create feed: %d %s", resp.StatusCode, body)
	}
	feed := decode[feeds.Feed](t, body)
	if feed.ID == 0 || feed.Title != "Example Blog" || feed.Unread != 1 {
		t.Fatalf("unexpected feed %+v", feed)
	}

	// Unparseable URL → 422.
	bad := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("<html>not a feed</html>"))
	}))
	defer bad.Close()
	resp, body = do(t, ts, "POST", "/api/feeds", token, strings.NewReader(`{"feed_url":"`+bad.URL+`/x"}`))
	if resp.StatusCode != 422 {
		t.Fatalf("bad feed status = %d, body=%s", resp.StatusCode, body)
	}

	// Refresh returns the new-item count after the feed is already fetched.
	resp, body = do(t, ts, "POST", "/api/feeds/"+itoa(feed.ID)+"/refresh", token, nil)
	if resp.StatusCode != 200 {
		t.Fatalf("refresh: %d %s", resp.StatusCode, body)
	}
	ref := decode[map[string]any](t, body)
	if ref["new_items"].(float64) != 0 {
		t.Fatalf("refresh new_items = %v", ref["new_items"])
	}

	// Items list has the one unread item, detail includes content.
	resp, body = do(t, ts, "GET", "/api/items", token, nil)
	if resp.StatusCode != 200 {
		t.Fatalf("items: %d %s", resp.StatusCode, body)
	}
	list := decode[struct {
		Items []feeds.Item `json:"items"`
		Total int          `json:"total"`
	}](t, body)
	if list.Total != 1 || len(list.Items) != 1 {
		t.Fatalf("list total=%d len=%d", list.Total, len(list.Items))
	}
	item := list.Items[0]
	if item.Content != "" || item.Summary != "" {
		t.Error("list items must not include content/summary")
	}
	resp, body = do(t, ts, "GET", "/api/items/"+itoa(item.ID), token, nil)
	detail := decode[feeds.Item](t, body)
	if detail.Summary != "sum" {
		t.Fatalf("detail summary = %q", detail.Summary)
	}

	// Mark read → unread count drops to 0.
	resp, _ = do(t, ts, "POST", "/api/items/"+itoa(item.ID)+"/read", token, nil)
	if resp.StatusCode != 204 {
		t.Fatalf("mark read status = %d", resp.StatusCode)
	}
	resp, body = do(t, ts, "GET", "/api/feeds/"+itoa(feed.ID), token, nil)
	if decode[feeds.Feed](t, body).Unread != 0 {
		t.Fatalf("unread after read: %s", body)
	}
}

func TestStatsAndReadAll(t *testing.T) {
	ts, repo := newTestServer(t)
	folder, _ := repo.CreateFolder("F")
	feed, _ := repo.CreateFeed(feeds.Feed{Title: "B", FeedURL: "https://b.example/rss", FolderID: &folder.ID})
	repo.AddItems(feed.ID, []feeds.Item{
		{GUID: "a", Title: "one"},
		{GUID: "b", Title: "two"},
	})

	resp, body := do(t, ts, "GET", "/api/stats", token, nil)
	stats := decode[map[string]any](t, body)
	if stats["feeds"].(float64) != 1 || stats["items"].(float64) != 2 || stats["unread"].(float64) != 2 {
		t.Fatalf("stats = %s", body)
	}

	resp, body = do(t, ts, "POST", "/api/items/read-all?folder_id="+itoa(folder.ID), token, nil)
	if resp.StatusCode != 200 {
		t.Fatalf("read-all: %d %s", resp.StatusCode, body)
	}
	if decode[map[string]any](t, body)["count"].(float64) != 2 {
		t.Fatalf("read-all body %s", body)
	}
	if _, body := do(t, ts, "GET", "/api/stats", token, nil); decode[map[string]any](t, body)["unread"].(float64) != 0 {
		t.Fatalf("unread after read-all: %s", body)
	}
}

func TestOPMLOverHTTP(t *testing.T) {
	ts, _ := newTestServer(t)

	var buf bytes.Buffer
	mw := multipart.NewWriter(&buf)
	fw, _ := mw.CreateFormFile("file", "subs.opml")
	fw.Write([]byte(`<?xml version="1.0"?><opml version="2.0"><body>
		<outline text="News"><outline type="rss" text="NYT" xmlUrl="https://www.nytimes.com/services/xml/rss/nyt/HomePage.xml"/></outline>
	</body></opml>`))
	mw.Close()

	req, _ := http.NewRequest("POST", ts.URL+"/api/opml/import", &buf)
	req.Header.Set("Content-Type", mw.FormDataContentType())
	req.Header.Set("Authorization", "Bearer "+token)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	data, _ := io.ReadAll(resp.Body)
	resp.Body.Close()
	if resp.StatusCode != 200 {
		t.Fatalf("import: %d %s", resp.StatusCode, data)
	}
	if decode[map[string]any](t, data)["added"].(float64) != 1 {
		t.Fatalf("import body %s", data)
	}

	resp, data = do(t, ts, "GET", "/api/opml/export", token, nil)
	if resp.StatusCode != 200 || resp.Header.Get("Content-Type") != "application/xml" {
		t.Fatalf("export: %d ct=%q", resp.StatusCode, resp.Header.Get("Content-Type"))
	}
	if !bytes.Contains(data, []byte("News")) || !bytes.Contains(data, []byte("xmlUrl")) {
		t.Fatalf("export body missing feeds: %s", data)
	}
}

func TestSetRenderMode(t *testing.T) {
	ts, repo := newTestServer(t)
	feed, _ := repo.CreateFeed(feeds.Feed{Title: "B", FeedURL: "https://b.example/rss"})

	resp, body := do(t, ts, "POST", "/api/feeds/"+itoa(feed.ID)+"/render-mode", token, strings.NewReader(`{"render_mode":2}`))
	if resp.StatusCode != 200 {
		t.Fatalf("set render-mode: %d %s", resp.StatusCode, body)
	}
	if decode[feeds.Feed](t, body).RenderMode != 2 {
		t.Fatalf("render_mode not persisted: %s", body)
	}

	// Invalid value → 400.
	if resp, _ := do(t, ts, "POST", "/api/feeds/"+itoa(feed.ID)+"/render-mode", token, strings.NewReader(`{"render_mode":3}`)); resp.StatusCode != 400 {
		t.Fatalf("invalid render_mode status = %d", resp.StatusCode)
	}
	// Missing feed → 404.
	if resp, _ := do(t, ts, "POST", "/api/feeds/99999/render-mode", token, strings.NewReader(`{"render_mode":1}`)); resp.StatusCode != 404 {
		t.Fatalf("missing feed status = %d", resp.StatusCode)
	}
}

func TestRenderEndpoint(t *testing.T) {
	ts, _ := newTestServer(t)

	page := `<html><head><title>Art</title></head><body><p>Hello article</p><img src="/pic.png"></body></html>`
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/err" {
			http.Error(w, "boom", http.StatusInternalServerError)
			return
		}
		_, _ = w.Write([]byte(page))
	}))
	defer upstream.Close()

	// Auth required.
	if resp, _ := do(t, ts, "GET", "/api/render?url="+url.QueryEscape(upstream.URL+"/art"), "", nil); resp.StatusCode != 401 {
		t.Fatalf("no-token status = %d", resp.StatusCode)
	}

	resp, body := do(t, ts, "GET", "/api/render?url="+url.QueryEscape(upstream.URL+"/art"), token, nil)
	if resp.StatusCode != 200 {
		t.Fatalf("render: %d %s", resp.StatusCode, body)
	}
	if ct := resp.Header.Get("Content-Type"); !strings.Contains(ct, "text/html") {
		t.Fatalf("content-type = %q", ct)
	}
	if csp := resp.Header.Get("Content-Security-Policy"); !strings.Contains(csp, "sandbox") {
		t.Fatalf("missing sandbox CSP: %q", csp)
	}
	if !strings.Contains(string(body), `<base href="`+upstream.URL+`/art">`) {
		t.Fatalf("base not injected: %s", body)
	}
	if !strings.Contains(string(body), "Hello article") {
		t.Fatalf("page body missing: %s", body)
	}

	// Non-http scheme → 400.
	if resp, _ := do(t, ts, "GET", "/api/render?url="+url.QueryEscape("file:///etc/passwd"), token, nil); resp.StatusCode != 400 {
		t.Fatalf("bad scheme status = %d", resp.StatusCode)
	}

	// Upstream 500 → 502.
	if resp, _ := do(t, ts, "GET", "/api/render?url="+url.QueryEscape(upstream.URL+"/err"), token, nil); resp.StatusCode != 502 {
		t.Fatalf("upstream error status = %d", resp.StatusCode)
	}
}

func TestNotFoundAndSPA(t *testing.T) {
	ts, _ := newTestServer(t)
	if resp, body := do(t, ts, "GET", "/api/feeds/999", token, nil); resp.StatusCode != 404 {
		t.Fatalf("missing feed status=%d body=%s", resp.StatusCode, body)
	}
	// SPA fallback serves index.html for any non-API route, and needs no token.
	if resp, body := do(t, ts, "GET", "/some/history/route", "", nil); resp.StatusCode != 200 || !bytes.Contains(body, []byte("tinyrss")) {
		t.Fatalf("spa fallback status=%d body=%s", resp.StatusCode, body)
	}
}

func itoa(n int) string { return strconv.Itoa(n) }
