package feeds

import (
	"strings"
	"testing"

	"github.com/mmcdole/gofeed"
)

func parseFixture(t *testing.T, xml string) *gofeed.Feed {
	t.Helper()
	pf, err := gofeed.NewParser().Parse(strings.NewReader(xml))
	if err != nil {
		t.Fatalf("parse fixture: %v", err)
	}
	return pf
}

func TestNormalizeRSS(t *testing.T) {
	items := normalizeItems(parseFixture(t, rssXML))
	if len(items) != 1 {
		t.Fatalf("want 1 item, got %d", len(items))
	}
	it := items[0]
	if it.Title != "Hello World" {
		t.Errorf("title = %q", it.Title)
	}
	if it.URL != "https://example.com/post/1" {
		t.Errorf("url = %q", it.URL)
	}
	if it.GUID != "guid-1" {
		t.Errorf("guid = %q", it.GUID)
	}
	if it.PublishedAt == "" {
		t.Error("published_at should be populated from pubDate")
	}
	if it.Summary == "" || it.Content == "" {
		t.Errorf("summary/content missing: %q / %q", it.Summary, it.Content)
	}
}

func TestNormalizeAtom(t *testing.T) {
	items := normalizeItems(parseFixture(t, atomXML))
	if len(items) != 1 {
		t.Fatalf("want 1 item, got %d", len(items))
	}
	it := items[0]
	if it.Title != "Atom Entry" {
		t.Errorf("title = %q", it.Title)
	}
	if it.GUID != "urn:uuid:1225c695-cfb8-4ebb-aaaa-80da344efa6a" {
		t.Errorf("guid = %q", it.GUID)
	}
	if it.Author != "Bob" {
		t.Errorf("author = %q", it.Author)
	}
	if it.PublishedAt == "" {
		t.Error("published_at should be populated from updated")
	}
	if !strings.Contains(it.Content, "bold") {
		t.Errorf("content = %q", it.Content)
	}
}

func TestNormalizeMissingFields(t *testing.T) {
	pf := parseFixture(t, `<rss version="2.0"><channel><item>
		<title>No metadata</title>
		<link>https://example.com/x</link>
	</item></channel></rss>`)
	items := normalizeItems(pf)
	if len(items) != 1 {
		t.Fatalf("want 1 item, got %d", len(items))
	}
	it := items[0]
	// No guid or date: fall back to link and now, respectively.
	if it.GUID != "https://example.com/x" {
		t.Errorf("guid fallback = %q", it.GUID)
	}
	if it.PublishedAt == "" {
		t.Error("published_at should fall back to now")
	}
}
