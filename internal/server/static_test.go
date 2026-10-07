package server

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"testing/fstest"
)

func staticTestServer(t *testing.T) *httptest.Server {
	t.Helper()
	spa := fstest.MapFS{
		"index.html":           &fstest.MapFile{Data: []byte("<h1>tinyrss</h1>")},
		"manifest.webmanifest": &fstest.MapFile{Data: []byte(`{"name":"TinyRSS"}`)},
		"sw.js":                &fstest.MapFile{Data: []byte("self.addEventListener('install', () => {})")},
		"icon.svg":             &fstest.MapFile{Data: []byte("<svg xmlns=\"http://www.w3.org/2000/svg\"/>")},
		"assets/app-abc123.js": &fstest.MapFile{Data: []byte("export default 1")},
	}
	ts, _ := newTestServerWithSPA(t, spa)
	return ts
}

func TestStaticAssets(t *testing.T) {
	ts := staticTestServer(t)
	cases := []struct {
		path        string
		contentType string
		cache       string
	}{
		{"/manifest.webmanifest", "application/manifest+json", "no-cache"},
		{"/sw.js", "", "no-cache"},
		{"/", "text/html; charset=utf-8", "no-cache"},
		{"/some/history/route", "text/html; charset=utf-8", "no-cache"},
		{"/assets/app-abc123.js", "", "public, max-age=31536000, immutable"},
		{"/icon.svg", "image/svg+xml", "public, max-age=3600"},
	}
	for _, tc := range cases {
		resp, body := do(t, ts, http.MethodGet, tc.path, "", nil)
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("GET %s status=%d body=%s", tc.path, resp.StatusCode, body)
		}
		if tc.contentType != "" && resp.Header.Get("Content-Type") != tc.contentType {
			t.Errorf("GET %s content-type=%q want %q", tc.path, resp.Header.Get("Content-Type"), tc.contentType)
		}
		if got := resp.Header.Get("Cache-Control"); got != tc.cache {
			t.Errorf("GET %s cache-control=%q want %q", tc.path, got, tc.cache)
		}
	}
}
