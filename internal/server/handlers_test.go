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
	"time"

	"tinyrss/internal/feeds"
	"tinyrss/internal/repository"
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

func newTestServer(t *testing.T) (*httptest.Server, *repository.Repo) {
	t.Helper()
	st, err := store.Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	repo := repository.NewRepo(st.DB)
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

	resp, body := do(t, ts, "POST", "/api/folders", token, strings.NewReader(`{"name":"Tech"}`))
	if resp.StatusCode != 200 {
		t.Fatalf("create folder: %d %s", resp.StatusCode, body)
	}
	folder := decode[repository.Folder](t, body)
	if folder.ID == 0 || folder.Name != "Tech" {
		t.Fatalf("unexpected folder %+v", folder)
	}

	resp, body = do(t, ts, "POST", "/api/feeds", token, strings.NewReader(`{"feed_url":"`+feedSrv.URL+`/feed"}`))
	if resp.StatusCode != 200 {
		t.Fatalf("create feed: %d %s", resp.StatusCode, body)
	}
	feed := decode[repository.Feed](t, body)
	// No content is fetched on create: the title falls back to the URL and the
	// feed has no items yet.
	if feed.ID == 0 || feed.Title != feedSrv.URL+"/feed" || feed.Unread != 0 {
		t.Fatalf("unexpected feed %+v", feed)
	}

	bad := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("<html>not a feed</html>"))
	}))
	defer bad.Close()
	// An unverified URL is stored as-is; create performs no fetch.
	resp, body = do(t, ts, "POST", "/api/feeds", token, strings.NewReader(`{"feed_url":"`+bad.URL+`/x"}`))
	if resp.StatusCode != 200 {
		t.Fatalf("unverified feed create status = %d, body=%s", resp.StatusCode, body)
	}

	// A manual refresh ingests the item.
	resp, body = do(t, ts, "POST", "/api/feeds/"+itoa(feed.ID)+"/refresh", token, nil)
	if resp.StatusCode != 200 {
		t.Fatalf("refresh: %d %s", resp.StatusCode, body)
	}
	ref := decode[map[string]any](t, body)
	if ref["new_items"].(float64) != 1 {
		t.Fatalf("refresh new_items = %v", ref["new_items"])
	}

	// Items list has the one unread item, detail includes content.
	resp, body = do(t, ts, "GET", "/api/items", token, nil)
	if resp.StatusCode != 200 {
		t.Fatalf("items: %d %s", resp.StatusCode, body)
	}
	list := decode[struct {
		Items []repository.Item `json:"items"`
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
	detail := decode[repository.Item](t, body)
	if detail.Summary != "sum" {
		t.Fatalf("detail summary = %q", detail.Summary)
	}

	resp, _ = do(t, ts, "POST", "/api/items/"+itoa(item.ID)+"/read", token, nil)
	if resp.StatusCode != 204 {
		t.Fatalf("mark read status = %d", resp.StatusCode)
	}
	resp, body = do(t, ts, "GET", "/api/feeds/"+itoa(feed.ID), token, nil)
	if decode[repository.Feed](t, body).Unread != 0 {
		t.Fatalf("unread after read: %s", body)
	}

	// Creating a feed with a folder_id attaches it to that group.
	resp, body = do(t, ts, "POST", "/api/feeds", token, strings.NewReader(
		`{"feed_url":"`+feedSrv.URL+`/feed2","folder_id":`+itoa(folder.ID)+`}`))
	if resp.StatusCode != 200 {
		t.Fatalf("create feed in folder: %d %s", resp.StatusCode, body)
	}
	folderFeed := decode[repository.Feed](t, body)
	if folderFeed.FolderID == nil || *folderFeed.FolderID != folder.ID {
		t.Fatalf("feed folder_id = %v, want %d", folderFeed.FolderID, folder.ID)
	}
}

func TestUpdateFeed(t *testing.T) {
	ts, _ := newTestServer(t)
	feedSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/rss+xml")
		_, _ = w.Write([]byte(testRSS))
	}))
	defer feedSrv.Close()

	resp, body := do(t, ts, "POST", "/api/feeds", token, strings.NewReader(`{"feed_url":"`+feedSrv.URL+`/feed"}`))
	if resp.StatusCode != 200 {
		t.Fatalf("create feed: %d %s", resp.StatusCode, body)
	}
	feed := decode[repository.Feed](t, body)

	resp, body = do(t, ts, "POST", "/api/folders", token, strings.NewReader(`{"name":"Down"}`))
	if resp.StatusCode != 200 {
		t.Fatalf("create folder: %d %s", resp.StatusCode, body)
	}
	folder := decode[repository.Folder](t, body)

	resp, body = do(t, ts, "PUT", "/api/feeds/"+itoa(feed.ID), token, strings.NewReader(
		`{"title":"Renamed","site_url":"https://news.example/","description":"new desc","folder_id":`+itoa(folder.ID)+`}`))
	if resp.StatusCode != 200 {
		t.Fatalf("update feed: %d %s", resp.StatusCode, body)
	}
	updated := decode[repository.Feed](t, body)
	if updated.Title != "Renamed" || updated.SiteURL != "https://news.example/" ||
		updated.Description != "new desc" || updated.FolderID == nil || *updated.FolderID != folder.ID {
		t.Fatalf("unexpected updated feed %+v", updated)
	}

	// Change the URL (distinct path) and clear the group; probe must succeed.
	altered := `{"title":"Renamed","feed_url":"` + feedSrv.URL + `/feed2","site_url":"https://site2.example/","description":"d2","folder_id":null}`
	resp, body = do(t, ts, "PUT", "/api/feeds/"+itoa(feed.ID), token, strings.NewReader(altered))
	if resp.StatusCode != 200 {
		t.Fatalf("update feed url: %d %s", resp.StatusCode, body)
	}
	updated = decode[repository.Feed](t, body)
	if updated.FeedURL != feedSrv.URL+"/feed2" || updated.SiteURL != "https://site2.example/" || updated.Description != "d2" {
		t.Fatalf("unexpected url update %+v", updated)
	}
	if updated.FolderID != nil {
		t.Fatalf("folder_id should be NULL, got %v", updated.FolderID)
	}

	// Re-assign the folder, then a description-only PUT must leave the omitted
	// title and folder_id untouched.
	resp, body = do(t, ts, "PUT", "/api/feeds/"+itoa(feed.ID), token, strings.NewReader(`{"folder_id":`+itoa(folder.ID)+`}`))
	if resp.StatusCode != 200 {
		t.Fatalf("re-assign folder: %d %s", resp.StatusCode, body)
	}
	resp, body = do(t, ts, "PUT", "/api/feeds/"+itoa(feed.ID), token, strings.NewReader(`{"description":"d3"}`))
	if resp.StatusCode != 200 {
		t.Fatalf("description-only update: %d %s", resp.StatusCode, body)
	}
	updated = decode[repository.Feed](t, body)
	if updated.Title != "Renamed" || updated.FolderID == nil || *updated.FolderID != folder.ID || updated.Description != "d3" {
		t.Fatalf("omitted title/folder_id regressed: %+v", updated)
	}

	bad := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("<html>not a feed</html>"))
	}))
	defer bad.Close()
	resp, body = do(t, ts, "PUT", "/api/feeds/"+itoa(feed.ID), token, strings.NewReader(`{"title":"Renamed","feed_url":"`+bad.URL+`/x"}`))
	if resp.StatusCode != 200 {
		t.Fatalf("changed feed url status = %d, body=%s", resp.StatusCode, body)
	}
	if decode[repository.Feed](t, body).FeedURL != bad.URL+"/x" {
		t.Fatalf("feed_url not stored as-is: %s", body)
	}

	resp, body = do(t, ts, "POST", "/api/feeds", token, strings.NewReader(`{"feed_url":"`+feedSrv.URL+`/feed3"}`))
	if resp.StatusCode != 200 {
		t.Fatalf("create second feed: %d %s", resp.StatusCode, body)
	}
	resp, body = do(t, ts, "PUT", "/api/feeds/"+itoa(feed.ID), token, strings.NewReader(`{"title":"Renamed","feed_url":"`+feedSrv.URL+`/feed3"}`))
	if resp.StatusCode != 409 {
		t.Fatalf("duplicate feed url status = %d, body=%s", resp.StatusCode, body)
	}
}

func TestDiscoverFeed(t *testing.T) {
	ts, _ := newTestServer(t)
	feedSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/rss+xml")
		_, _ = w.Write([]byte(testRSS))
	}))
	defer feedSrv.Close()

	resp, body := do(t, ts, "POST", "/api/feeds/discover", token, strings.NewReader(`{"url":"`+feedSrv.URL+`/feed"}`))
	if resp.StatusCode != 200 {
		t.Fatalf("discover: %d %s", resp.StatusCode, body)
	}
	d := decode[feeds.Discovered](t, body)
	if d.Title != "Example Blog" || d.SiteURL != "https://example.com/" || d.Description != "desc" {
		t.Fatalf("unexpected discover result %+v", d)
	}
	if d.FeedURL != feedSrv.URL+"/feed" {
		t.Fatalf("feed_url = %q", d.FeedURL)
	}

	bad := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("<html>not a feed</html>"))
	}))
	defer bad.Close()
	if resp, _ := do(t, ts, "POST", "/api/feeds/discover", token, strings.NewReader(`{"url":"`+bad.URL+`/x"}`)); resp.StatusCode != 422 {
		t.Fatalf("bad discover status = %d", resp.StatusCode)
	}
}

func TestStatsAndReadAll(t *testing.T) {
	ts, repo := newTestServer(t)
	folder, _ := repo.CreateFolder("F")
	feed, _ := repo.CreateFeed(repository.Feed{Title: "B", FeedURL: "https://b.example/rss", FolderID: &folder.ID})
	repo.AddItems(feed.ID, []repository.Item{
		{GUID: "a", Title: "one"},
		{GUID: "b", Title: "two"},
	})

	resp, body := do(t, ts, "GET", "/api/stats", token, nil)
	stats := decode[map[string]any](t, body)
	if stats["feeds"].(float64) != 1 || stats["items"].(float64) != 2 || stats["unread"].(float64) != 2 {
		t.Fatalf("stats = %s", body)
	}

	resp, body = do(t, ts, "POST", "/api/items/read-all?feed_id="+itoa(feed.ID), token, nil)
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
	feed, _ := repo.CreateFeed(repository.Feed{Title: "B", FeedURL: "https://b.example/rss"})

	resp, body := do(t, ts, "POST", "/api/feeds/"+itoa(feed.ID)+"/render-mode", token, strings.NewReader(`{"render_mode":2}`))
	if resp.StatusCode != 200 {
		t.Fatalf("set render-mode: %d %s", resp.StatusCode, body)
	}
	if decode[repository.Feed](t, body).RenderMode != 2 {
		t.Fatalf("render_mode not persisted: %s", body)
	}

	if resp, _ := do(t, ts, "POST", "/api/feeds/"+itoa(feed.ID)+"/render-mode", token, strings.NewReader(`{"render_mode":3}`)); resp.StatusCode != 400 {
		t.Fatalf("invalid render_mode status = %d", resp.StatusCode)
	}
	if resp, _ := do(t, ts, "POST", "/api/feeds/99999/render-mode", token, strings.NewReader(`{"render_mode":1}`)); resp.StatusCode != 404 {
		t.Fatalf("missing feed status = %d", resp.StatusCode)
	}
}

func TestRenderEndpoint(t *testing.T) {
	ts, _ := newTestServer(t)

	page := `<html><head><title>Art</title><script>window.evil=1</script></head>
		<body><p onclick="evil()">Hello article</p><img src="/pic.png"></body></html>`
	var hits int
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits++
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
	if csp := resp.Header.Get("Content-Security-Policy"); strings.Contains(csp, "allow-scripts") {
		t.Fatalf("CSP must not allow scripts: %q", csp)
	}
	if !strings.Contains(string(body), `<base href="`+upstream.URL+`/art">`) {
		t.Fatalf("base not injected: %s", body)
	}
	if !strings.Contains(string(body), "Hello article") {
		t.Fatalf("page body missing: %s", body)
	}
	if !strings.Contains(string(body), `class="tinyrss-article"`) {
		t.Fatalf("custom styled wrapper missing: %s", body)
	}
	if strings.Contains(string(body), "<script") || strings.Contains(string(body), "onclick") {
		t.Fatalf("scripts not stripped from rendered page: %s", body)
	}

	// A second render of the same URL is served from the DB cache.
	do(t, ts, "GET", "/api/render?url="+url.QueryEscape(upstream.URL+"/art"), token, nil)
	if hits != 1 {
		t.Fatalf("upstream hits = %d, want 1 (cached)", hits)
	}

	if resp, _ := do(t, ts, "GET", "/api/render?url="+url.QueryEscape("file:///etc/passwd"), token, nil); resp.StatusCode != 400 {
		t.Fatalf("bad scheme status = %d", resp.StatusCode)
	}

	if resp, _ := do(t, ts, "GET", "/api/render?url="+url.QueryEscape(upstream.URL+"/err"), token, nil); resp.StatusCode != 502 {
		t.Fatalf("upstream error status = %d", resp.StatusCode)
	}
}

func TestRefreshAllProgressEndpoints(t *testing.T) {
	ts, repo := newTestServer(t)
	feedSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(testRSS))
	}))
	defer feedSrv.Close()
	if _, err := repo.CreateFeed(repository.Feed{Title: "F", FeedURL: feedSrv.URL + "/feed"}); err != nil {
		t.Fatal(err)
	}

	resp, body := do(t, ts, "POST", "/api/refresh", token, nil)
	if resp.StatusCode != 200 {
		t.Fatalf("POST /api/refresh status = %d, body=%s", resp.StatusCode, body)
	}
	_ = decode[feeds.RefreshJob](t, body)

	// Poll until the job finishes; progress then reports the completed job.
	var job feeds.RefreshJob
	for i := 0; i < 200; i++ {
		resp, body = do(t, ts, "GET", "/api/refresh/progress", token, nil)
		if resp.StatusCode != 200 {
			t.Fatalf("progress status = %d", resp.StatusCode)
		}
		job = decode[feeds.RefreshJob](t, body)
		if !job.Running {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	if job.Total != 1 || job.Done != 1 || job.Failed != 0 {
		t.Fatalf("job = %+v, want total=1 done=1 failed=0", job)
	}
}

func TestSettingsAndFetchLogsEndpoints(t *testing.T) {
	ts, repo := newTestServer(t)
	feed, _ := repo.CreateFeed(repository.Feed{Title: "B", FeedURL: "https://b.example/rss"})
	if _, err := repo.DB.Exec(`INSERT INTO fetch_logs(feed_id, success, error) VALUES (?, 0, ?)`,
		feed.ID, "upstream returned 500"); err != nil {
		t.Fatal(err)
	}

	resp, body := do(t, ts, "GET", "/api/settings", token, nil)
	if resp.StatusCode != 200 {
		t.Fatalf("get settings: %d %s", resp.StatusCode, body)
	}
	if s := decode[repository.Settings](t, body); s.FetchLogCleanupSeconds != 30*86400 || s.RenderCacheCleanupSeconds != 30*86400 {
		t.Fatalf("default settings = %+v, want both 30*86400", s)
	}
	// Partial updates apply only the present field and keep the other.
	resp, body = do(t, ts, "PUT", "/api/settings", token, strings.NewReader(`{"fetch_log_cleanup_seconds":604800}`))
	if resp.StatusCode != 200 {
		t.Fatalf("put settings: %d", resp.StatusCode)
	}
	if s := decode[repository.Settings](t, body); s.FetchLogCleanupSeconds != 604800 || s.RenderCacheCleanupSeconds != 30*86400 {
		t.Fatalf("settings after fetch update = %+v", s)
	}
	resp, body = do(t, ts, "PUT", "/api/settings", token, strings.NewReader(`{"render_cache_cleanup_seconds":1209600}`))
	if resp.StatusCode != 200 {
		t.Fatalf("put render settings: %d", resp.StatusCode)
	}
	if s := decode[repository.Settings](t, body); s.FetchLogCleanupSeconds != 604800 || s.RenderCacheCleanupSeconds != 1209600 {
		t.Fatalf("settings after render update = %+v", s)
	}
	if resp, _ = do(t, ts, "PUT", "/api/settings", token, strings.NewReader(`{"fetch_log_cleanup_seconds":-1}`)); resp.StatusCode != 400 {
		t.Fatalf("negative fetch seconds status = %d, want 400", resp.StatusCode)
	}
	if resp, _ = do(t, ts, "PUT", "/api/settings", token, strings.NewReader(`{"render_cache_cleanup_seconds":-1}`)); resp.StatusCode != 400 {
		t.Fatalf("negative render seconds status = %d, want 400", resp.StatusCode)
	}

	// Refresh interval & gap: default, persist valid values, reject invalid ones.
	resp, body = do(t, ts, "GET", "/api/settings", token, nil)
	if resp.StatusCode != 200 {
		t.Fatalf("get settings: %d %s", resp.StatusCode, body)
	}
	if s := decode[repository.Settings](t, body); s.RefreshIntervalSeconds != 900 {
		t.Fatalf("default refresh interval = %d, want 900", s.RefreshIntervalSeconds)
	}
	if s := decode[repository.Settings](t, body); s.MinRefreshGapSeconds != 600 {
		t.Fatalf("default min refresh gap = %d, want 600", s.MinRefreshGapSeconds)
	}
	resp, body = do(t, ts, "PUT", "/api/settings", token, strings.NewReader(`{"refresh_interval_seconds":30}`))
	if resp.StatusCode != 200 {
		t.Fatalf("put refresh interval: %d %s", resp.StatusCode, body)
	}
	if s := decode[repository.Settings](t, body); s.RefreshIntervalSeconds != 30 {
		t.Fatalf("refresh interval after update = %d, want 30", s.RefreshIntervalSeconds)
	}
	if resp, _ = do(t, ts, "PUT", "/api/settings", token, strings.NewReader(`{"refresh_interval_seconds":0}`)); resp.StatusCode != 400 {
		t.Fatalf("zero refresh interval status = %d, want 400", resp.StatusCode)
	}
	resp, body = do(t, ts, "PUT", "/api/settings", token, strings.NewReader(`{"min_refresh_gap_seconds":60}`))
	if resp.StatusCode != 200 {
		t.Fatalf("put min refresh gap: %d %s", resp.StatusCode, body)
	}
	if s := decode[repository.Settings](t, body); s.MinRefreshGapSeconds != 60 {
		t.Fatalf("min refresh gap after update = %d, want 60", s.MinRefreshGapSeconds)
	}
	if resp, _ = do(t, ts, "PUT", "/api/settings", token, strings.NewReader(`{"min_refresh_gap_seconds":0}`)); resp.StatusCode != 400 {
		t.Fatalf("zero min refresh gap status = %d, want 400", resp.StatusCode)
	}

	resp, body = do(t, ts, "GET", "/api/fetch-logs", token, nil)
	if resp.StatusCode != 200 {
		t.Fatalf("fetch-logs: %d %s", resp.StatusCode, body)
	}
	logs := decode[[]repository.FetchLog](t, body)
	if len(logs) != 1 || logs[0].Success || logs[0].FeedTitle != "B" || logs[0].Error == "" {
		t.Fatalf("fetch logs = %+v", logs)
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

func TestFeedStatsEndpoint(t *testing.T) {
	ts, repo := newTestServer(t)
	feed, err := repo.CreateFeed(repository.Feed{Title: "A", FeedURL: "https://a.example/rss"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := repo.AddItems(feed.ID, []repository.Item{
		{GUID: "a1", Title: "t", PublishedAt: "2024-05-01 10:00:00"},
	}); err != nil {
		t.Fatal(err)
	}

	type statsBody struct {
		Days  int                   `json:"days"`
		Feeds []repository.FeedStat `json:"feeds"`
	}

	resp, body := do(t, ts, "GET", "/api/stats/feeds", token, nil)
	if resp.StatusCode != 200 {
		t.Fatalf("status=%d body=%s", resp.StatusCode, body)
	}
	out := decode[statsBody](t, body)
	if out.Days != 30 {
		t.Fatalf("default days = %d, want 30", out.Days)
	}
	found := false
	for _, s := range out.Feeds {
		if s.FeedID == feed.ID && s.Total == 1 && len(s.Series) == 1 && s.LatestAt == "2024-05-01 10:00:00" {
			found = true
		}
	}
	if !found {
		t.Fatalf("feed A not found with expected stats: %+v", out.Feeds)
	}

	// days param is clamped to [1, 365].
	resp, body = do(t, ts, "GET", "/api/stats/feeds?days=9999", token, nil)
	if resp.StatusCode != 200 {
		t.Fatalf("clamp-high status=%d", resp.StatusCode)
	}
	if d := decode[statsBody](t, body).Days; d != 365 {
		t.Fatalf("clamped-high days = %d, want 365", d)
	}
	resp, body = do(t, ts, "GET", "/api/stats/feeds?days=0", token, nil)
	if resp.StatusCode != 200 {
		t.Fatalf("clamp-low status=%d", resp.StatusCode)
	}
	if d := decode[statsBody](t, body).Days; d != 1 {
		t.Fatalf("clamped-low days = %d, want 1", d)
	}
}

func itoa(n int) string { return strconv.Itoa(n) }
