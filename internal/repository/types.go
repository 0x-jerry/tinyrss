// Package repository owns the SQLite-backed persistence for the reader domain:
// the item/feed/folder repository, fetch logs, render cache, settings, and OPML
// import/export, plus the data model those tables map to. HTTP lives in
// internal/server and fetching/render in internal/feeds.
package repository

const TimeLayout = "2006-01-02 15:04:05"

// The default settings, expressed in seconds. The auto-clean retentions (0
// disables cleanup) and the refresh interval/gap are all user-adjustable via
// the settings API. Gap/interval defaults are exported for the fetcher's
// fallback when the settings row is absent or unreadable.
const (
	defaultFetchLogCleanupSeconds    = 30 * 86400
	defaultRenderCacheCleanupSeconds = 30 * 86400
	DefaultRefreshIntervalSeconds    = 15 * 60
	DefaultMinRefreshGapSeconds      = 10 * 60
)

type Folder struct {
	ID        int    `json:"id"`
	Name      string `json:"name"`
	SortOrder int    `json:"sort_order"`
}

type Feed struct {
	ID            int    `json:"id"`
	Title         string `json:"title"`
	FeedURL       string `json:"feed_url"`
	SiteURL       string `json:"site_url"`
	Description   string `json:"description"`
	ProxyURL      string `json:"proxy_url"` // empty = direct connection
	RenderMode    int    `json:"render_mode"` // 0=content, 1=iframe site view
	FolderID      *int   `json:"folder_id"`
	ETag          string `json:"etag"`
	LastModified  string `json:"last_modified"`
	LastFetchedAt string `json:"last_fetched_at"`
	FetchError    string `json:"fetch_error"`
	Unread        int    `json:"unread"`
	CreatedAt     string `json:"created_at"`
	UpdatedAt     string `json:"updated_at"`
}

// FetchLog is one recorded fetch attempt: whether it succeeded, the error (if
// any), and when it ran.
type FetchLog struct {
	ID        int    `json:"id"`
	FeedID    int    `json:"feed_id"`
	FeedTitle string `json:"feed_title"`
	Success   bool   `json:"success"`
	Error     string `json:"error"`
	FetchedAt string `json:"fetched_at"`
}

// Settings holds the user-adjustable app settings. FetchLogCleanupSeconds is
// the auto-clean retention for fetch logs and RenderCacheCleanupSeconds the
// retention for the server-render cache, both in seconds; 0 disables cleanup.
// RefreshIntervalSeconds is the auto-refresh poll interval and
// MinRefreshGapSeconds the minimum gap between manual refresh-alls.
type Settings struct {
	FetchLogCleanupSeconds    int `json:"fetch_log_cleanup_seconds"`
	RenderCacheCleanupSeconds int `json:"render_cache_cleanup_seconds"`
	RefreshIntervalSeconds    int `json:"refresh_interval_seconds"`
	MinRefreshGapSeconds      int `json:"min_refresh_gap_seconds"`
}

// Item is both the list and detail representation. Summary/Content are only
// populated on detail requests and omitted (omitempty) from list responses.
type Item struct {
	ID          int    `json:"id"`
	FeedID      int    `json:"feed_id"`
	FeedTitle   string `json:"feed_title"`
	GUID        string `json:"-"`
	Title       string `json:"title"`
	URL         string `json:"url"`
	Author      string `json:"author"`
	Summary     string `json:"summary,omitempty"`
	Content     string `json:"content,omitempty"`
	PublishedAt string `json:"published_at"`
	IsRead      bool   `json:"is_read"`
	IsStarred  bool   `json:"is_starred"`
	CreatedAt   string `json:"created_at"`
}

// ItemFilter describes the article list query. Zero/unset values are ignored.
type ItemFilter struct {
	FeedID  int
	Unread  bool
	Starred bool
	Search  string
	Page    int
	Limit   int
}
