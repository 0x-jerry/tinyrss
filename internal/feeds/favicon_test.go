package feeds

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestFetchFavicon(t *testing.T) {
	var gotPath, gotQuery string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotQuery = r.URL.RawQuery
		w.Header().Set("Content-Type", "image/png")
		_, _ = w.Write([]byte("PNGDATA"))
	}))
	defer srv.Close()

	restore := SetFaviconBaseURL(srv.URL + "/s2/favicons")
	defer restore()

	f := NewFetcher(nil)
	ct, body, err := f.FetchFavicon(context.Background(), "example.com", 32)
	if err != nil {
		t.Fatalf("FetchFavicon: %v", err)
	}
	if ct != "image/png" {
		t.Fatalf("content type = %q, want image/png", ct)
	}
	if string(body) != "PNGDATA" {
		t.Fatalf("body = %q", body)
	}
	if !strings.Contains(gotQuery, "domain=example.com") {
		t.Fatalf("query %q missing domain", gotQuery)
	}
	if !strings.Contains(gotQuery, "sz=32") {
		t.Fatalf("query %q missing sz=32", gotQuery)
	}
	if gotPath != "/s2/favicons" {
		t.Fatalf("path = %q", gotPath)
	}
}

func TestFetchFaviconUpstreamError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "nope", http.StatusInternalServerError)
	}))
	defer srv.Close()
	restore := SetFaviconBaseURL(srv.URL)
	defer restore()

	f := NewFetcher(nil)
	if _, _, err := f.FetchFavicon(context.Background(), "example.com", 32); err == nil {
		t.Fatal("expected error on non-200 upstream")
	}
}

func TestFetchFaviconCapsBody(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "image/png")
		_, _ = io.CopyN(w, strings.NewReader(strings.Repeat("x", maxFaviconSize*2)), int64(maxFaviconSize*2))
	}))
	defer srv.Close()
	restore := SetFaviconBaseURL(srv.URL)
	defer restore()

	f := NewFetcher(nil)
	_, body, err := f.FetchFavicon(context.Background(), "example.com", 32)
	if err != nil {
		t.Fatalf("FetchFavicon: %v", err)
	}
	if len(body) != maxFaviconSize {
		t.Fatalf("body len = %d, want %d", len(body), maxFaviconSize)
	}
}
