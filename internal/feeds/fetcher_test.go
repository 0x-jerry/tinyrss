package feeds

import (
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
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
