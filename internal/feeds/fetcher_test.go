package feeds

import (
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"
)

func TestFetcherConditionalGETAndNewItems(t *testing.T) {
	const etag = `"v1"`
	var mu sync.Mutex
	requests := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		requests++
		mu.Unlock()
		if r.Header.Get("If-None-Match") == etag {
			w.WriteHeader(http.StatusNotModified)
			return
		}
		w.Header().Set("ETag", etag)
		_, _ = w.Write([]byte(rssXML))
	}))
	defer srv.Close()

	repo := newTestRepo(t)
	feed, err := repo.CreateFeed(Feed{Title: "F", FeedURL: srv.URL + "/feed.xml"})
	if err != nil {
		t.Fatal(err)
	}
	fetcher := NewFetcher(repo)

	// First fetch parses items; all are new.
	newCount, err := fetcher.RefreshFeed(feed.ID)
	if err != nil {
		t.Fatalf("first refresh: %v", err)
	}
	if newCount != 1 {
		t.Fatalf("newCount = %d, want 1", newCount)
	}
	feed, _ = repo.GetFeed(feed.ID)
	if feed.ETag != etag {
		t.Errorf("stored etag = %q, want %q", feed.ETag, etag)
	}
	if feed.LastFetchedAt == "" {
		t.Error("last_fetched_at should be set after fetch")
	}

	// Conditional GET hits the 304 path: no re-parse, zero new items.
	newCount, err = fetcher.RefreshFeed(feed.ID)
	if err != nil {
		t.Fatalf("second refresh: %v", err)
	}
	if newCount != 0 {
		t.Fatalf("newCount on 304 = %d, want 0", newCount)
	}
	mu.Lock()
	got := requests
	mu.Unlock()
	if got != 2 {
		t.Fatalf("requests = %d, want 2 (initial + conditional)", got)
	}
}

func TestFetcherRecordsErrorKeepsFeedAlive(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	repo := newTestRepo(t)
	feed, err := repo.CreateFeed(Feed{Title: "F", FeedURL: srv.URL + "/feed.xml"})
	if err != nil {
		t.Fatal(err)
	}
	fetcher := NewFetcher(repo)
	if _, err := fetcher.RefreshFeed(feed.ID); err == nil {
		t.Fatal("expected refresh error on 500")
	}
	// Feed stays alive with an error recorded for retry.
	feed, err = repo.GetFeed(feed.ID)
	if err != nil {
		t.Fatal(err)
	}
	if feed.FetchError == "" {
		t.Error("fetch_error should be recorded on failure")
	}
	if feed.LastFetchedAt == "" {
		t.Error("last_fetched_at should be touched so retry cadence is known")
	}
}

func TestFetcherRecordsFetchLogsOnSuccessAndFailure(t *testing.T) {
	ok := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(rssXML))
	}))
	defer ok.Close()
	bad := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer bad.Close()

	repo := newTestRepo(t)
	fetcher := NewFetcher(repo)
	good, _ := repo.CreateFeed(Feed{Title: "Good", FeedURL: ok.URL + "/feed.xml"})
	broken, _ := repo.CreateFeed(Feed{Title: "Broken", FeedURL: bad.URL + "/feed.xml"})

	if _, err := fetcher.RefreshFeed(good.ID); err != nil {
		t.Fatalf("refresh good feed: %v", err)
	}
	if _, err := fetcher.RefreshFeed(broken.ID); err == nil {
		t.Fatalf("refresh broken feed should fail")
	}

	logs, err := repo.ListFetchLogs(0)
	if err != nil {
		t.Fatal(err)
	}
	if len(logs) != 2 {
		t.Fatalf("fetch logs = %+v, want 2 rows", logs)
	}
	// Newest (failed) first.
	if logs[0].Success || logs[0].Error == "" || logs[0].FeedTitle != "Broken" {
		t.Fatalf("newest log = %+v, want failed Broken", logs[0])
	}
	if !logs[1].Success || logs[1].Error != "" || logs[1].FeedTitle != "Good" {
		t.Fatalf("older log = %+v, want success Good", logs[1])
	}
}

// waitIdle polls Progress until the refresh-all job is no longer running.
func waitIdle(t *testing.T, f *Fetcher) RefreshJob {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for {
		job := f.Progress()
		if !job.Running {
			return job
		}
		if time.Now().After(deadline) {
			t.Fatal("refresh-all job did not finish in time")
		}
		time.Sleep(10 * time.Millisecond)
	}
}

func TestRefreshAllGapSkipsRecentFeeds(t *testing.T) {
	var mu sync.Mutex
	requests := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		requests++
		mu.Unlock()
		_, _ = w.Write([]byte(rssXML))
	}))
	defer srv.Close()

	repo := newTestRepo(t)
	fetcher := NewFetcher(repo)
	feed, err := repo.CreateFeed(Feed{Title: "F", FeedURL: srv.URL + "/feed.xml"})
	if err != nil {
		t.Fatal(err)
	}

	// A fetch within minRefreshGap means a manual refresh-all skips it.
	if _, err := fetcher.RefreshFeed(feed.ID); err != nil {
		t.Fatal(err)
	}
	if err := fetcher.RefreshAllAsync(); err != nil {
		t.Fatal(err)
	}
	job := waitIdle(t, fetcher)
	if job.Total != 1 || job.Done != 1 || job.Failed != 0 {
		t.Fatalf("job = %+v, want total=1 done=1 failed=0", job)
	}
	mu.Lock()
	got := requests
	mu.Unlock()
	if got != 1 {
		t.Fatalf("requests = %d, want 1 (recent feed skipped by gap)", got)
	}

	// Backdating last_fetched_at beyond the gap lets the next run fetch again.
	if _, err := repo.DB.Exec(`UPDATE feeds SET last_fetched_at = ? WHERE id = ?`,
		time.Now().Add(-2*minRefreshGap).Format(time.RFC3339), feed.ID); err != nil {
		t.Fatal(err)
	}
	if err := fetcher.RefreshAllAsync(); err != nil {
		t.Fatal(err)
	}
	job = waitIdle(t, fetcher)
	if job.Done != 1 {
		t.Fatalf("job = %+v, want done=1", job)
	}
	mu.Lock()
	got = requests
	mu.Unlock()
	if got != 2 {
		t.Fatalf("requests = %d, want 2 (aged feed refreshed)", got)
	}
}
