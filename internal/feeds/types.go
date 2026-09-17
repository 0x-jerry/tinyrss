// Package feeds owns the reader domain: the item/feed/folder repository,
// fetching & parsing, scheduling, and OPML. HTTP lives in internal/server.
package feeds

const TimeLayout = "2006-01-02 15:04:05"

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
	FeedID   int
	FolderID int
	Unread   bool
	Starred  bool
	Search   string
	Page     int
	Limit    int
}

type ErrNotFound struct{ what string }

func (e ErrNotFound) Error() string { return e.what + " not found" }

type ErrConflict struct{ what string }

func (e ErrConflict) Error() string { return e.what }
