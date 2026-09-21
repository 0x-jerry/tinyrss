package feeds

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	"golang.org/x/net/html"
)

// Discovered is the metadata the "Detect" button pulls back from a URL: the
// resolved feed URL plus fields the add/edit dialogs auto-fill.
type Discovered struct {
	FeedURL     string `json:"feed_url"`
	Title       string `json:"title"`
	SiteURL     string `json:"site_url"`
	Description string `json:"description"`
}

// feedLinkTypes are the HTML link rel=alternate types that identify a feed; a
// rel=alternate link with no type is tried as a last resort.
var feedLinkTypes = map[string]bool{
	"application/rss+xml":  true,
	"application/atom+xml": true,
	"application/rdf+xml":  true,
}

// Discover resolves a single URL into feed metadata: the URL is parsed directly
// when it is itself a feed, otherwise the page's HTML is scanned for a feed
// link (autodiscovery). It never writes anything — dialogs use it only to
// prefill the form.
func (f *Fetcher) Discover(raw string) (*Discovered, error) {
	u, err := url.Parse(raw)
	if err != nil || (u.Scheme != "http" && u.Scheme != "https") {
		return nil, errors.New("url must be http(s)")
	}
	if d, err := f.discoverDirect(raw); err == nil {
		return d, nil
	}
	return f.discoverFromHTML(raw)
}

// discoverDirect parses raw as RSS/Atom and returns its metadata.
func (f *Fetcher) discoverDirect(raw string) (*Discovered, error) {
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

// discoverFromHTML fetches raw as a web page, collects its feed-link hrefs and
// probes each candidate until one parses as a feed.
func (f *Fetcher) discoverFromHTML(raw string) (*Discovered, error) {
	req, err := http.NewRequest(http.MethodGet, raw, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", userAgent)
	resp, err := f.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("upstream returned %s", resp.Status)
	}
	pageURL, _ := url.Parse(raw)
	for _, href := range feedHrefs(io.LimitReader(resp.Body, maxBodySize)) {
		linkURL, err := url.Parse(href)
		if err != nil {
			continue
		}
		cand := pageURL.ResolveReference(linkURL).String()
		pf, err := f.Probe(cand)
		if err != nil {
			continue
		}
		return &Discovered{
			FeedURL:     cand,
			Title:       pf.Title,
			SiteURL:     pf.Link,
			Description: pf.Description,
		}, nil
	}
	return nil, fmt.Errorf("no feed found at %s", raw)
}

// feedHrefs returns the hrefs of a page's feed <link rel=alternate> elements,
// typed feed links first, then untagged ones as a fallback.
func feedHrefs(r io.Reader) []string {
	doc, err := html.Parse(r)
	if err != nil {
		return nil
	}
	var typed, untyped []string
	var walk func(*html.Node)
	walk = func(n *html.Node) {
		if n.Type == html.ElementNode && n.Data == "link" {
			if !relHasAlternate(linkAttr(n, "rel")) {
				return
			}
			href := linkAttr(n, "href")
			typ := strings.ToLower(linkAttr(n, "type"))
			if href == "" {
				return
			}
			if feedLinkTypes[typ] {
				typed = append(typed, href)
			} else if typ == "" {
				untyped = append(untyped, href)
			}
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			walk(c)
		}
	}
	walk(doc)
	return append(typed, untyped...)
}

func linkAttr(n *html.Node, key string) string {
	for _, a := range n.Attr {
		if strings.EqualFold(a.Key, key) {
			return a.Val
		}
	}
	return ""
}

func relHasAlternate(rel string) bool {
	for _, r := range strings.Fields(rel) {
		if strings.EqualFold(r, "alternate") {
			return true
		}
	}
	return false
}
