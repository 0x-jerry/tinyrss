package feeds

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestDiscoverDirectFeedURL(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(rssXML))
	}))
	defer srv.Close()

	f := NewFetcher(newTestRepo(t))
	d, err := f.Discover(srv.URL + "/feed.xml")
	if err != nil {
		t.Fatalf("discover direct feed: %v", err)
	}
	if d.FeedURL != srv.URL+"/feed.xml" {
		t.Errorf("FeedURL = %q, want %q", d.FeedURL, srv.URL+"/feed.xml")
	}
	if d.Title != "Example Blog" {
		t.Errorf("Title = %q", d.Title)
	}
	if d.SiteURL != "https://example.com/" {
		t.Errorf("SiteURL = %q", d.SiteURL)
	}
	if d.Description != "A demo blog" {
		t.Errorf("Description = %q", d.Description)
	}
}

func TestDiscoverAutodiscoverAbsolute(t *testing.T) {
	var srv *httptest.Server
	srv = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/feed.xml" {
			_, _ = w.Write([]byte(rssXML))
			return
		}
		_, _ = w.Write([]byte(`<!doctype html><html><head>
			<link rel="alternate" type="application/rss+xml" href="` + srv.URL + `/feed.xml">
		</head><body>home</body></html>`))
	}))
	defer srv.Close()

	f := NewFetcher(newTestRepo(t))
	d, err := f.Discover(srv.URL + "/")
	if err != nil {
		t.Fatalf("autodiscover: %v", err)
	}
	if d.FeedURL != srv.URL+"/feed.xml" {
		t.Errorf("FeedURL = %q, want %q", d.FeedURL, srv.URL+"/feed.xml")
	}
	if d.Title != "Example Blog" {
		t.Errorf("Title = %q", d.Title)
	}
}

func TestDiscoverAutodiscoverRelative(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/blog/feed.xml":
			_, _ = w.Write([]byte(rssXML))
		case "/blog/index.html":
			_, _ = w.Write([]byte(`<!doctype html><html><head>
				<link rel="alternate" type="application/atom+xml" href="feed.xml">
			</head></html>`))
		}
	}))
	defer srv.Close()

	f := NewFetcher(newTestRepo(t))
	d, err := f.Discover(srv.URL + "/blog/index.html")
	if err != nil {
		t.Fatalf("autodiscover relative: %v", err)
	}
	if d.FeedURL != srv.URL+"/blog/feed.xml" {
		t.Errorf("FeedURL = %q, want %q", d.FeedURL, srv.URL+"/blog/feed.xml")
	}
}

func TestDiscoverNoFeedLinks(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`<!doctype html><html><head>
			<link rel="stylesheet" href="/x.css">
		</head><body>nothing here</body></html>`))
	}))
	defer srv.Close()

	f := NewFetcher(newTestRepo(t))
	if _, err := f.Discover(srv.URL + "/"); err == nil {
		t.Fatal("expected error for a page without feed links")
	}
}
