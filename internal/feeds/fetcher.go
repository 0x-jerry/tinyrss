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

	// minRefreshGap throttles a manual refresh-all so feeds fetched within this
	// window are skipped — a fixed ceiling keeps hammering the button from
	// re-hitting recently fetched feeds. Make configurable only if needed.
	minRefreshGap = 10 * time.Minute
)

// RefreshJob is a point-in-time snapshot of the background refresh-all job.
// Total counts every feed considered (including ones skipped by minRefreshGap);
// Done counts processed (success + failed).
type RefreshJob struct {
	Running      bool   `json:"running"`
	Total        int    `json:"total"`
	Done         int    `json:"done"`
	Failed       int    `json:"failed"`
	NewItems     int    `json:"new_items"`
	CurrentID    int    `json:"current_feed_id"`
	CurrentTitle string `json:"current_feed_title"`
}

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
	jobMu   sync.Mutex
	job     RefreshJob
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
				f.maybePruneFetchLogs()
				f.refreshDue(interval)
			case <-f.stopCh:
				return
			}
		}
	}()
	f.maybePruneFetchLogs()
}

// maybePruneFetchLogs deletes logs older than the configured retention; a
// retention of 0 (disabled) or a settings read error skips cleanup.
func (f *Fetcher) maybePruneFetchLogs() {
	s, err := f.repo.GetSettings()
	if err != nil || s.FetchLogCleanupDays <= 0 {
		return
	}
	_, _ = f.repo.PruneFetchLogs(time.Now().AddDate(0, 0, -s.FetchLogCleanupDays))
}

func (f *Fetcher) Stop() {
	if f.stopCh != nil {
		close(f.stopCh)
	}
	f.stopWg.Wait()
	f.refreshWg.Wait()
}

// lastFetched parses a feed's recorded fetch time; zero means never fetched.
// Reading a DATETIME column through the sqlite driver yields RFC3339; older
// rows may hold "2006-01-02 15:04:05". Both are UTC instants, so parse and
// compare against the now-based cutoff regardless of the local timezone.
func lastFetched(fd Feed) time.Time {
	s := fd.LastFetchedAt
	if s == "" {
		return time.Time{}
	}
	if t, err := time.Parse(time.RFC3339Nano, s); err == nil {
		return t
	}
	if t, err := time.ParseInLocation(TimeLayout, s, time.UTC); err == nil {
		return t
	}
	return time.Time{}
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
		last := lastFetched(fd)
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

// RefreshAllAsync kicks off a background refresh-all and returns immediately.
// Feeds fetched within minRefreshGap are skipped so repeated clicks don't
// re-hit them. Progress is observable via Progress. It is a no-op (nil) when a
// refresh is already running, so the caller can keep polling the active job.
func (f *Fetcher) RefreshAllAsync() error {
	f.jobMu.Lock()
	if f.job.Running {
		f.jobMu.Unlock()
		return nil
	}
	f.jobMu.Unlock()

	feeds, err := f.repo.ListFeeds()
	if err != nil {
		return err
	}

	f.jobMu.Lock()
	f.job = RefreshJob{Running: true, Total: len(feeds)}
	f.jobMu.Unlock()

	f.refreshWg.Add(1)
	go f.runRefreshAll(feeds)
	return nil
}

func (f *Fetcher) runRefreshAll(feeds []Feed) {
	defer f.refreshWg.Done()
	defer func() {
		f.jobMu.Lock()
		f.job.Running = false
		f.job.CurrentID = 0
		f.job.CurrentTitle = ""
		f.jobMu.Unlock()
	}()

	cutoff := time.Now().Add(-minRefreshGap)
	var wg sync.WaitGroup
	for _, fd := range feeds {
		if last := lastFetched(fd); !last.IsZero() && !last.Before(cutoff) {
			f.markDone(0, nil)
			continue
		}
		wg.Add(1)
		go func(fd Feed) {
			defer wg.Done()
			f.sem <- struct{}{}
			defer func() { <-f.sem }()
			// Set current from inside the worker so the progress bar tracks a feed
			// that is actually being fetched, not the last one merely dispatched.
			f.setCurrent(fd)
			newItems, err := f.RefreshFeed(fd.ID)
			f.markDone(newItems, err)
		}(fd)
	}
	wg.Wait()
}

func (f *Fetcher) setCurrent(fd Feed) {
	f.jobMu.Lock()
	f.job.CurrentID = fd.ID
	f.job.CurrentTitle = fd.Title
	f.jobMu.Unlock()
}

func (f *Fetcher) markDone(newItems int, err error) {
	f.jobMu.Lock()
	defer f.jobMu.Unlock()
	f.job.Done++
	if err != nil {
		f.job.Failed++
		return
	}
	f.job.NewItems += newItems
}

// Progress returns a snapshot of the in-flight or last refresh-all job.
func (f *Fetcher) Progress() RefreshJob {
	f.jobMu.Lock()
	defer f.jobMu.Unlock()
	return f.job
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
