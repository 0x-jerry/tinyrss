package feeds

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/mmcdole/gofeed"
	"golang.org/x/sync/singleflight"
)

const (
	maxBodySize = 32 << 20 // cap fetched body: a runaway feed can't OOM us
	workerCount = 4
	userAgent   = "tinyrss/1.0"
)

// Fetcher pulls feeds with conditional GET, dedupes concurrent refreshes per
// feed via singleflight, and runs a ticker that refreshes due feeds with a
// bounded worker pool.
type Fetcher struct {
	repo    *Repo
	client  *http.Client
	sf      singleflight.Group
	sem     chan struct{}
	stopCh  chan struct{}
	stopWg  sync.WaitGroup
	refreshWg sync.WaitGroup
}

func NewFetcher(repo *Repo) *Fetcher {
	client := &http.Client{
		Timeout: 30 * time.Second,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if len(via) >= 5 {
				return errors.New("too many redirects")
			}
			return nil
		},
	}
	return &Fetcher{
		repo:   repo,
		client: client,
		sem:    make(chan struct{}, workerCount),
	}
}

// Start launches the periodic refresh loop; Stop shuts it down gracefully.
// RefreshFeed and RefreshAll remain callable without ever calling Start.
func (f *Fetcher) Start(interval time.Duration) {
	f.stopCh = make(chan struct{})
	f.stopWg.Add(1)
	go func() {
		defer f.stopWg.Done()
		t := time.NewTicker(interval)
		defer t.Stop()
		for {
			select {
			case <-t.C:
				f.refreshDue(interval)
			case <-f.stopCh:
				return
			}
		}
	}()
}

func (f *Fetcher) Stop() {
	if f.stopCh != nil {
		close(f.stopCh)
	}
	f.stopWg.Wait()
	f.refreshWg.Wait()
}

// refreshDue refreshes every feed that has never been fetched or was last
// fetched more than interval ago, with a bounded worker pool.
func (f *Fetcher) refreshDue(interval time.Duration) {
	feeds, err := f.repo.ListFeeds()
	if err != nil {
		return
	}
	cutoff := time.Now().Add(-interval)
	due := make([]int, 0, len(feeds))
	for _, fd := range feeds {
		var last time.Time
		if fd.LastFetchedAt != "" {
			if t, err := time.ParseInLocation(TimeLayout, fd.LastFetchedAt, time.Local); err == nil {
				last = t
			}
		}
		if last.IsZero() || last.Before(cutoff) {
			due = append(due, fd.ID)
		}
	}
	var wg sync.WaitGroup
	for _, id := range due {
		f.refreshWg.Add(1)
		wg.Add(1)
		go func(id int) {
			defer f.refreshWg.Done()
			defer wg.Done()
			f.sem <- struct{}{}
			defer func() { <-f.sem }()
			_, _ = f.RefreshFeed(id)
		}(id)
	}
	wg.Wait()
}

// RefreshFeed refreshes one feed, blocking until done. Concurrent calls for
// the same feed coalesce via singleflight. Returns the count of new items.
func (f *Fetcher) RefreshFeed(id int) (int, error) {
	v, err, _ := f.sf.Do(strconv.Itoa(id), func() (any, error) {
		return f.doRefresh(id)
	})
	if err != nil {
		return 0, err
	}
	return v.(int), nil
}

// RefreshAll refreshes every feed, returning how many refreshed without error.
func (f *Fetcher) RefreshAll() int {
	feeds, err := f.repo.ListFeeds()
	if err != nil {
		return 0
	}
	var mu sync.Mutex
	var ok int
	var wg sync.WaitGroup
	for _, fd := range feeds {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			f.sem <- struct{}{}
			defer func() { <-f.sem }()
			if _, err := f.RefreshFeed(id); err == nil {
				mu.Lock()
				ok++
				mu.Unlock()
			}
		}(fd.ID)
	}
	wg.Wait()
	return ok
}

// Probe downloads and parses a feed URL with no stored etag — used when adding
// a feed, to validate the URL and capture metadata. Unparseable URLs error.
func (f *Fetcher) Probe(url string) (*gofeed.Feed, error) {
	req, err := http.NewRequest(http.MethodGet, url, nil)
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
	pf, err := gofeed.NewParser().Parse(io.LimitReader(resp.Body, maxBodySize))
	if err != nil {
		return nil, err
	}
	return pf, nil
}

func (f *Fetcher) doRefresh(id int) (int, error) {
	feed, err := f.repo.GetFeed(id)
	if err != nil {
		return 0, err
	}
	req, err := http.NewRequest(http.MethodGet, feed.FeedURL, nil)
	if err != nil {
		return 0, err
	}
	if feed.ETag != "" {
		req.Header.Set("If-None-Match", feed.ETag)
	}
	if feed.LastModified != "" {
		req.Header.Set("If-Modified-Since", feed.LastModified)
	}
	req.Header.Set("User-Agent", userAgent)

	resp, err := f.client.Do(req)
	if err != nil {
		_ = f.repo.recordFetchResult(id, "", "", err.Error(), false)
		return 0, err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotModified {
		_ = f.repo.recordFetchResult(id, feed.ETag, feed.LastModified, "", true)
		return 0, nil
	}
	if resp.StatusCode != http.StatusOK {
		msg := fmt.Sprintf("upstream returned %s", resp.Status)
		_ = f.repo.recordFetchResult(id, "", "", msg, false)
		return 0, errors.New(msg)
	}

	pf, err := gofeed.NewParser().Parse(io.LimitReader(resp.Body, maxBodySize))
	if err != nil {
		msg := "feed did not parse as RSS/Atom: " + err.Error()
		_ = f.repo.recordFetchResult(id, "", "", msg, false)
		return 0, err
	}
	items := normalizeItems(pf)
	newCount, err := f.repo.AddItems(id, items)
	if err != nil {
		return 0, err
	}
	etag := resp.Header.Get("ETag")
	lastMod := resp.Header.Get("Last-Modified")
	_ = f.repo.recordFetchResult(id, etag, lastMod, "", true)
	return newCount, nil
}

// normalizeItems maps a gofeed.Feed into our Item rows. GUID falls back to
// link/title; malformed or missing dates fall back to now.
func normalizeItems(pf *gofeed.Feed) []Item {
	now := time.Now().UTC().Format(TimeLayout)
	items := make([]Item, 0, len(pf.Items))
	for _, it := range pf.Items {
		guid := it.GUID
		if guid == "" {
			guid = it.Link
		}
		if guid == "" {
			guid = it.Title
		}
		pub := it.PublishedParsed
		if pub == nil {
			pub = it.UpdatedParsed
		}
		published := now
		if pub != nil {
			published = pub.UTC().Format(TimeLayout)
		}
		author := ""
		if it.Author != nil {
			author = it.Author.Name
		}
		items = append(items, Item{
			GUID:        guid,
			Title:       strings.TrimSpace(it.Title),
			URL:         it.Link,
			Author:      author,
			Summary:     it.Description,
			Content:     it.Content,
			PublishedAt: published,
		})
	}
	return items
}
