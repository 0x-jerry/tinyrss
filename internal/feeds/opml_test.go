package feeds

import (
	"encoding/xml"
	"testing"
)

const opmlSample = `<?xml version="1.0" encoding="UTF-8"?>
<opml version="2.0">
<head><title>subscriptions</title></head>
<body>
<outline text="Tech">
  <outline type="rss" text="Go Blog" xmlUrl="https://go.dev/blog/feed.atom" htmlUrl="https://go.dev/blog"/>
  <outline type="rss" text="Ars Technica" xmlUrl="https://feeds.arstechnica.com/arstechnica/index" htmlUrl="https://arstechnica.com"/>
</outline>
<outline type="rss" text="Hacker News" xmlUrl="https://news.ycombinator.com/rss" htmlUrl="https://news.ycombinator.com"/>
</body>
</opml>`

func TestOPMLRoundTrip(t *testing.T) {
	repo := newTestRepo(t)

	added, err := repo.ImportOPML([]byte(opmlSample))
	if err != nil {
		t.Fatalf("import: %v", err)
	}
	if added != 3 {
		t.Fatalf("imported %d feeds, want 3", added)
	}

	// Re-importing is idempotent (feeds dedup by URL).
	if added, _ := repo.ImportOPML([]byte(opmlSample)); added != 0 {
		t.Fatalf("second import added %d, want 0", added)
	}

	out, err := repo.ExportOPML()
	if err != nil {
		t.Fatalf("export: %v", err)
	}
	var doc struct {
		Body struct {
			Outlines []struct {
				Text     string `xml:"text,attr"`
				XMLURL   string `xml:"xmlUrl,attr"`
				Outlines []struct {
					Text   string `xml:"text,attr"`
					XMLURL string `xml:"xmlUrl,attr"`
				} `xml:"outline"`
			} `xml:"outline"`
		} `xml:"body"`
	}
	if err := xml.Unmarshal(out, &doc); err != nil {
		t.Fatalf("parse export: %v", err)
	}

	// One top-level folder ("Tech") with 2 feeds, one flat feed, all present.
	byURL := map[string]bool{}
	for _, o := range doc.Body.Outlines {
		if o.XMLURL != "" {
			byURL[o.XMLURL] = true
			continue
		}
		if o.Text != "Tech" {
			t.Errorf("unexpected top-level outline %q", o.Text)
		}
		for _, c := range o.Outlines {
			if c.Text == "" || c.XMLURL == "" {
				t.Errorf("folder feed missing text/xmlUrl: %+v", c)
			}
			byURL[c.XMLURL] = true
		}
	}
	if len(byURL) != 3 {
		t.Fatalf("exported %d feeds, want 3: %v", len(byURL), byURL)
	}
	for _, want := range []string{
		"https://go.dev/blog/feed.atom",
		"https://feeds.arstechnica.com/arstechnica/index",
		"https://news.ycombinator.com/rss",
	} {
		if !byURL[want] {
			t.Errorf("export missing feed %q", want)
		}
	}
}

func TestOPMLImportUnparseable(t *testing.T) {
	repo := newTestRepo(t)
	if _, err := repo.ImportOPML([]byte("not xml")); err == nil {
		t.Fatal("expected parse error for invalid XML")
	}
}
