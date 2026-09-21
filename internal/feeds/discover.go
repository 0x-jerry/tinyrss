package feeds

import (
	"errors"
	"net/url"
)

// Discovered is the metadata the "Detect" button pulls back from a feed URL:
// the resolved feed URL plus fields the add/edit dialogs auto-fill.
type Discovered struct {
	FeedURL     string `json:"feed_url"`
	Title       string `json:"title"`
	SiteURL     string `json:"site_url"`
	Description string `json:"description"`
}

// Discover resolves a single URL into feed metadata by parsing it directly as
// RSS/Atom. It never writes anything — dialogs use it only to prefill the form.
// HTML autodiscovery for homepage URLs is a later issue.
func (f *Fetcher) Discover(raw string) (*Discovered, error) {
	u, err := url.Parse(raw)
	if err != nil || (u.Scheme != "http" && u.Scheme != "https") {
		return nil, errors.New("url must be http(s)")
	}
	pf, err := f.Probe(raw)
	if err != nil {
		return nil, err
	}
	return &Discovered{
		FeedURL:     raw,
		Title:       pf.Title,
		SiteURL:     pf.Link,
		Description: pf.Description,
	}, nil
}
