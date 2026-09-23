// Package feeds is the reader's fetch & render service: it polls feeds into
// the repository with conditional GET, coalesces concurrent refreshes, and
// serves extracted article renders. Persistence and the data model live in
// internal/repository; HTTP in internal/server.
package feeds

import (
	"context"
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

	"tinyrss/internal/repository"
)

const (
	maxBodySize   = 32 << 20 // cap fetched body: a runaway feed can't OOM us
	workerCount   = 4
	renderWorkers = 2
	userAgent   = "tinyrss/1.0"
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
	repo       *repository.Repo
	client     *http.Client
	ctx        context.Context
	cancel     context.CancelFunc
	sf         singleflight.Group
	renderSf   singleflight.Group
	sem        chan struct{}
	renderSem  chan struct{}
	proxyPool  *clientPool
	renderPool *clientPool
	stopCh     chan struct{}
	startOnce  sync.Once
	stopOnce   sync.Once
	stopWg     sync.WaitGroup
	refreshWg  sync.WaitGroup
	jobMu      sync.Mutex
	job        RefreshJob
}

func NewFetcher(repo *repository.Repo) *Fetcher {
	ctx, cancel := context.WithCancel(context.Background())
	return &Fetcher{
		repo:       repo,
		client:     newHTTPClient(baseTransport()),
		ctx:        ctx,
		cancel:     cancel,
		sem:        make(chan struct{}, workerCount),
		renderSem:  make(chan struct{}, renderWorkers),
		proxyPool:  newClientPool(),
		renderPool: newClientPool(),
	}
}

// Start launches the periodic refresh loop; Stop shuts it down gracefully.
// The poll interval comes from the settings table each cycle (falling back to
// the repository default), so a Settings change takes effect on the next tick.
// RefreshFeed and RefreshAll remain callable without ever calling Start.
func (f *Fetcher) Start() {
	f.startOnce.Do(func() {
		f.stopCh = make(chan struct{})
		f.stopWg.Go(func() {
			for {
				current := f.refreshInterval()
				select {
				case <-time.After(current):
					f.maybePruneFetchLogs()
					f.maybePruneRenderCache()
					f.refreshDue(current)
				case <-f.stopCh:
					return
				}
			}
		})
		f.maybePruneFetchLogs()
		f.maybePruneRenderCache()
	})
}

// refreshInterval returns the configured auto-refresh poll interval, falling
// back to the repository default when unset or unreadable.
func (f *Fetcher) refreshInterval() time.Duration {
	if s, err := f.repo.GetSettings(); err == nil && s.RefreshIntervalSeconds >= 1 {
		return time.Duration(s.RefreshIntervalSeconds) * time.Second
	}
	return repository.DefaultRefreshIntervalSeconds * time.Second
}

// minRefreshGap returns the configured throttle for a manual refresh-all:
// feeds fetched within this window are skipped, keeping a mashed refresh
// button from re-hitting them. Falls back to the repository default.
func (f *Fetcher) minRefreshGap() time.Duration {
	if s, err := f.repo.GetSettings(); err == nil && s.MinRefreshGapSeconds >= 1 {
		return time.Duration(s.MinRefreshGapSeconds) * time.Second
	}
	return repository.DefaultMinRefreshGapSeconds * time.Second
}

// maybePruneFetchLogs deletes logs older than the configured retention; a
// retention of 0 (disabled) or a settings read error skips cleanup.
func (f *Fetcher) maybePruneFetchLogs() {
	s, err := f.repo.GetSettings()
	if err != nil || s.FetchLogCleanupSeconds <= 0 {
		return
	}
	_, _ = f.repo.PruneFetchLogs(time.Now().Add(-time.Duration(s.FetchLogCleanupSeconds) * time.Second))
}

// maybePruneRenderCache expires cached server-rendered articles older than the
// configured retention; a retention of 0 (disabled) or a settings read error
// skips cleanup.
func (f *Fetcher) maybePruneRenderCache() {
	s, err := f.repo.GetSettings()
	if err != nil || s.RenderCacheCleanupSeconds <= 0 {
		return
	}
	_, _ = f.repo.PruneRenderCache(time.Now().Add(-time.Duration(s.RenderCacheCleanupSeconds) * time.Second))
}

func (f *Fetcher) Stop() {
	f.stopOnce.Do(func() {
		f.cancel()
		if f.stopCh != nil {
			close(f.stopCh)
		}
	})
	f.stopWg.Wait()
	f.refreshWg.Wait()
	f.proxyPool.closeIdle()
	f.renderPool.closeIdle()
}

// lastFetchedAt parses a feed's recorded fetch time; zero means never fetched.
// Reading a DATETIME column through the sqlite driver yields RFC3339; older
// rows may hold "2006-01-02 15:04:05". Both are UTC instants, so parse and
// compare against the now-based cutoff regardless of the local timezone.
func lastFetchedAt(s string) time.Time {
	if s == "" {
		return time.Time{}
	}
	if t, err := time.Parse(time.RFC3339Nano, s); err == nil {
		return t
	}
	if t, err := time.ParseInLocation(repository.TimeLayout, s, time.UTC); err == nil {
		return t
	}
	return time.Time{}
}

// refreshDue refreshes every feed that has never been fetched or was last
// fetched more than interval ago, with a bounded worker pool.
func (f *Fetcher) refreshDue(interval time.Duration) {
	refs, err := f.repo.ListFeedRefs()
	if err != nil {
		return
	}
	cutoff := time.Now().Add(-interval)
	due := make([]int, 0, len(refs))
	for _, ref := range refs {
		last := lastFetchedAt(ref.LastFetchedAt)
		if last.IsZero() || last.Before(cutoff) {
			due = append(due, ref.ID)
		}
	}
	runPool(f, due, func(id int) {
		_, _ = f.RefreshFeed(id)
	})
}

// runPool consumes items with at most workerCount goroutines, so feed count
// never translates into goroutine or in-flight-buffer growth.
func runPool[T any](f *Fetcher, items []T, fn func(T)) {
	workers := min(workerCount, len(items))
	if workers == 0 {
		return
	}
	jobs := make(chan T)
	var wg sync.WaitGroup
	for range workers {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for item := range jobs {
				f.sem <- struct{}{}
				fn(item)
				<-f.sem
			}
		}()
	}
	for _, item := range items {
		select {
		case jobs <- item:
		case <-f.ctx.Done():
			close(jobs)
			wg.Wait()
			return
		}
	}
	close(jobs)
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

	refs, err := f.repo.ListFeedRefs()
	if err != nil {
		return err
	}

	f.jobMu.Lock()
	f.job = RefreshJob{Running: true, Total: len(refs)}
	f.jobMu.Unlock()

	f.refreshWg.Add(1)
	go f.runRefreshAll(refs)
	return nil
}

func (f *Fetcher) runRefreshAll(refs []repository.FeedRef) {
	defer f.refreshWg.Done()
	defer func() {
		f.jobMu.Lock()
		f.job.Running = false
		f.job.CurrentID = 0
		f.job.CurrentTitle = ""
		f.jobMu.Unlock()
	}()

	cutoff := time.Now().Add(-f.minRefreshGap())
	due := make([]repository.FeedRef, 0, len(refs))
	for _, ref := range refs {
		if last := lastFetchedAt(ref.LastFetchedAt); !last.IsZero() && !last.Before(cutoff) {
			f.markDone(0, nil)
			continue
		}
		due = append(due, ref)
	}
	runPool(f, due, func(ref repository.FeedRef) {
		// Set current from inside the worker so the progress bar tracks a feed
		// that is actually being fetched, not the last one merely dispatched.
		f.setCurrent(ref)
		newItems, err := f.RefreshFeed(ref.ID)
		f.markDone(newItems, err)
	})
}

func (f *Fetcher) setCurrent(ref repository.FeedRef) {
	f.jobMu.Lock()
	f.job.CurrentID = ref.ID
	f.job.CurrentTitle = ref.Title
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
func (f *Fetcher) Probe(url, proxyURL string) (*gofeed.Feed, error) {
	client, err := f.clientFor(proxyURL)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(f.ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", userAgent)
	resp, err := client.Do(req)
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
	client, err := f.clientFor(feed.ProxyURL)
	if err != nil {
		_ = f.repo.RecordFetchResult(id, "", "", err.Error(), false)
		return 0, err
	}
	req, err := http.NewRequestWithContext(f.ctx, http.MethodGet, feed.FeedURL, nil)
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

	resp, err := client.Do(req)
	if err != nil {
		if f.ctx.Err() != nil {
			return 0, err
		}
		_ = f.repo.RecordFetchResult(id, "", "", err.Error(), false)
		return 0, err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotModified {
		_ = f.repo.RecordFetchResult(id, feed.ETag, feed.LastModified, "", true)
		return 0, nil
	}
	if resp.StatusCode != http.StatusOK {
		msg := fmt.Sprintf("upstream returned %s", resp.Status)
		_ = f.repo.RecordFetchResult(id, "", "", msg, false)
		return 0, errors.New(msg)
	}

	pf, err := gofeed.NewParser().Parse(io.LimitReader(resp.Body, maxBodySize))
	if err != nil {
		msg := "feed did not parse as RSS/Atom: " + err.Error()
		_ = f.repo.RecordFetchResult(id, "", "", msg, false)
		return 0, err
	}
	items := normalizeItems(pf)
	newCount, err := f.repo.AddItems(id, items)
	if err != nil {
		return 0, err
	}
	etag := resp.Header.Get("ETag")
	lastMod := resp.Header.Get("Last-Modified")
	_ = f.repo.RecordFetchResult(id, etag, lastMod, "", true)
	return newCount, nil
}

// normalizeItems maps a gofeed.Feed into our Item rows. GUID falls back to
// link/title; malformed or missing dates fall back to a fixed oldest time
// (1900-01-01 00:00:00).
func normalizeItems(pf *gofeed.Feed) []repository.Item {
	// The oldest representable value we store: items whose publish date could
	// not be parsed sort to the bottom of the newest-first list, not the top.
	const oldest = "1900-01-01 00:00:00"
	items := make([]repository.Item, 0, len(pf.Items))
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
		published := oldest
		if pub != nil {
			published = pub.UTC().Format(repository.TimeLayout)
		}
		author := ""
		if it.Author != nil {
			author = it.Author.Name
		}
		items = append(items, repository.Item{
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
