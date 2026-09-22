package repository

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"tinyrss/internal/store"
)

func newTestRepo(t *testing.T) *Repo {
	t.Helper()
	st, err := store.Open(":memory:")
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	t.Cleanup(func() { st.Close() })
	return NewRepo(st.DB)
}

func TestStoreMigrationsApplied(t *testing.T) {
	st, err := store.Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()
	var n int
	if err := st.DB.QueryRow(`SELECT COUNT(*) FROM schema_migrations`).Scan(&n); err != nil {
		t.Fatalf("schema_migrations: %v", err)
	}
	if n != 8 {
		t.Fatalf("want 8 applied migrations, got %d", n)
	}
}

func TestSetRenderMode(t *testing.T) {
	repo := newTestRepo(t)
	feed, err := repo.CreateFeed(Feed{Title: "B", FeedURL: "https://b.example/rss"})
	if err != nil {
		t.Fatal(err)
	}
	if feed.RenderMode != 0 {
		t.Fatalf("default render_mode = %d, want 0", feed.RenderMode)
	}
	got, err := repo.SetRenderMode(feed.ID, 1)
	if err != nil {
		t.Fatalf("set render mode: %v", err)
	}
	if got.RenderMode != 1 {
		t.Fatalf("render_mode after set = %d, want 1", got.RenderMode)
	}
	if _, err := repo.SetRenderMode(99999, 1); err == nil {
		t.Fatal("expected not-found for missing feed")
	}
}

func TestRepoCRUDDedupUnread(t *testing.T) {
	repo := newTestRepo(t)

	folder, err := repo.CreateFolder("Tech")
	if err != nil {
		t.Fatalf("create folder: %v", err)
	}
	feed, err := repo.CreateFeed(Feed{
		Title:    "Example Blog",
		FeedURL:  "https://example.com/rss",
		FolderID: &folder.ID,
	})
	if err != nil {
		t.Fatalf("create feed: %v", err)
	}

	first := []Item{
		{GUID: "a", Title: "One", URL: "https://example.com/1", Content: "c1"},
		{GUID: "b", Title: "Two", URL: "https://example.com/2", Content: "c2"},
	}
	if n, err := repo.AddItems(feed.ID, first); err != nil || n != 2 {
		t.Fatalf("first AddItems: n=%d err=%v", n, err)
	}
	if n, err := repo.AddItems(feed.ID, first); err != nil || n != 0 {
		t.Fatalf("dup AddItems: n=%d err=%v", n, err)
	}

	feed, err = repo.GetFeed(feed.ID)
	if err != nil {
		t.Fatal(err)
	}
	if feed.Unread != 2 {
		t.Fatalf("unread = %d, want 2", feed.Unread)
	}

	items, total, err := repo.ListItems(ItemFilter{FeedID: feed.ID})
	if err != nil {
		t.Fatal(err)
	}
	if total != 2 {
		t.Fatalf("total = %d, want 2", total)
	}
	if err := repo.SetRead(items[0].ID, true); err != nil {
		t.Fatal(err)
	}
	feed, _ = repo.GetFeed(feed.ID)
	if feed.Unread != 1 {
		t.Fatalf("unread after read = %d, want 1", feed.Unread)
	}

	count, err := repo.MarkAllRead(ItemFilter{FeedID: feed.ID})
	if err != nil || count != 2 {
		t.Fatalf("mark-all-read: count=%d err=%v", count, err)
	}
	if feed, _ = repo.GetFeed(feed.ID); feed.Unread != 0 {
		t.Fatalf("unread after mark-all = %d, want 0", feed.Unread)
	}

	// Deleting a folder folds its feeds to uncategorized.
	if err := repo.DeleteFolder(folder.ID); err != nil {
		t.Fatal(err)
	}
	feed, _ = repo.GetFeed(feed.ID)
	if feed.FolderID != nil {
		t.Fatalf("folder_id should be NULL after delete, got %v", feed.FolderID)
	}
}

func TestRepoSearch(t *testing.T) {
	repo := newTestRepo(t)
	feed, err := repo.CreateFeed(Feed{Title: "F", FeedURL: "https://example.com/rss"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := repo.AddItems(feed.ID, []Item{
		{GUID: "a", Title: "Golang generics guide", Content: "deep dive"},
		{GUID: "b", Title: "Rust borrow checker", Content: "ownership"},
	}); err != nil {
		t.Fatal(err)
	}
	// Search hits only the matching item (title + content indexed).
	items, total, err := repo.ListItems(ItemFilter{Search: "golang"})
	if err != nil {
		t.Fatal(err)
	}
	if total != 1 || items[0].Title != "Golang generics guide" {
		t.Fatalf("search total=%d items=%+v", total, items)
	}
}

func TestUpdateFeedPartial(t *testing.T) {
	repo := newTestRepo(t)
	folderA, err := repo.CreateFolder("A")
	if err != nil {
		t.Fatal(err)
	}
	folderB, err := repo.CreateFolder("B")
	if err != nil {
		t.Fatal(err)
	}
	feed, err := repo.CreateFeed(Feed{Title: "Original", FeedURL: "https://example.com/rss", FolderID: &folderA.ID})
	if err != nil {
		t.Fatal(err)
	}

	title := "Renamed"
	feed, err = repo.UpdateFeed(feed.ID, &title, nil, nil, nil, FolderField{})
	if err != nil {
		t.Fatal(err)
	}
	if feed.Title != "Renamed" || feed.FolderID == nil || *feed.FolderID != folderA.ID {
		t.Fatalf("after title-only update = %+v", feed)
	}

	feed, err = repo.UpdateFeed(feed.ID, nil, nil, nil, nil, FolderField{Null: true})
	if err != nil {
		t.Fatal(err)
	}
	if feed.FolderID != nil {
		t.Fatalf("folder should be cleared, got %v", *feed.FolderID)
	}
	if feed.Title != "Renamed" {
		t.Fatalf("title regressed to %q", feed.Title)
	}

	feed, err = repo.UpdateFeed(feed.ID, nil, nil, nil, nil, FolderField{Set: true, ID: folderB.ID})
	if err != nil {
		t.Fatal(err)
	}
	if feed.FolderID == nil || *feed.FolderID != folderB.ID {
		t.Fatalf("folder should be B, got %v", feed.FolderID)
	}

	if _, err := repo.UpdateFeed(feed.ID, nil, nil, nil, nil, FolderField{Set: true, ID: 99999}); err == nil {
		t.Fatal("expected not-found for missing folder")
	}

	feed2, err := repo.UpdateFeed(feed.ID, nil, nil, nil, nil, FolderField{})
	if err != nil {
		t.Fatal(err)
	}
	if feed2.Title != feed.Title || feed2.FolderID == nil || *feed2.FolderID != folderB.ID {
		t.Fatalf("no-op update changed the feed: %+v", feed2)
	}
}

func TestFolderFieldUnmarshal(t *testing.T) {
	decode := func(s string) (FolderField, error) {
		var f FolderField
		err := json.Unmarshal([]byte(s), &f)
		return f, err
	}
	if f, err := decode("null"); err != nil || !f.Null || f.Set {
		t.Fatalf("null -> %+v err=%v", f, err)
	}
	if f, err := decode("3"); err != nil || f.Set != true || f.Null || f.ID != 3 {
		t.Fatalf("3 -> %+v err=%v", f, err)
	}
	if _, err := decode(`"abc"`); err == nil {
		t.Fatal("expected error for non-numeric value")
	}
}

func TestPruneFetchLogsUTC(t *testing.T) {
	repo := newTestRepo(t)
	feed, err := repo.CreateFeed(Feed{Title: "F", FeedURL: "https://example.com/rss"})
	if err != nil {
		t.Fatal(err)
	}
	// Stored fetched_at values are UTC wall-clock, as CURRENT_TIMESTAMP writes.
	for _, ts := range []string{"2020-01-14 22:00:00", "2020-01-15 02:00:00"} {
		if _, err := repo.DB.Exec(
			`INSERT INTO fetch_logs(feed_id, success, error, fetched_at) VALUES (?, 1, '', ?)`,
			feed.ID, ts); err != nil {
			t.Fatal(err)
		}
	}
	// A cutoff expressed in a non-UTC zone must be normalised to UTC before the
	// string comparison. UTC+8 local 2020-01-15 08:00 is UTC 2020-01-15 00:00,
	// which prunes only the 22:00 row and keeps the 02:00 row. Without the UTC
	// normalisation the literal would be the local "2020-01-15 08:00:00" and
	// both rows would be pruned.
	cutoff := time.Date(2020, 1, 15, 8, 0, 0, 0, time.FixedZone("UTC+8", 8*3600))
	pruned, err := repo.PruneFetchLogs(cutoff)
	if err != nil {
		t.Fatal(err)
	}
	if pruned != 1 {
		t.Fatalf("pruned = %d, want 1", pruned)
	}
	logs, err := repo.ListFetchLogs(0)
	if err != nil {
		t.Fatal(err)
	}
	if len(logs) != 1 || len(logs[0].FetchedAt) < 10 || logs[0].FetchedAt[:10] != "2020-01-15" {
		t.Fatalf("surviving log = %+v, want the 2020-01-15 row", logs)
	}
}

func TestPruneRenderCacheUTC(t *testing.T) {
	repo := newTestRepo(t)
	for _, row := range []struct{ url, ts string }{
		{"https://example.com/a", "2020-01-14 22:00:00"},
		{"https://example.com/b", "2020-01-15 02:00:00"},
	} {
		if _, err := repo.DB.Exec(
			`INSERT INTO render_cache(url, html, created_at) VALUES (?, '<p>x</p>', ?)`,
			row.url, row.ts); err != nil {
			t.Fatal(err)
		}
	}
	// UTC+8 local 2020-01-15 08:00 is UTC 2020-01-15 00:00; only the older row
	// (22:00 on the 14th) is older than the cutoff.
	cutoff := time.Date(2020, 1, 15, 8, 0, 0, 0, time.FixedZone("UTC+8", 8*3600))
	pruned, err := repo.PruneRenderCache(cutoff)
	if err != nil {
		t.Fatal(err)
	}
	if pruned != 1 {
		t.Fatalf("pruned = %d, want 1", pruned)
	}
	if _, ok, _ := repo.GetRenderCache("https://example.com/b"); !ok {
		t.Fatal("newer cached render was pruned")
	}
	if _, ok, _ := repo.GetRenderCache("https://example.com/a"); ok {
		t.Fatal("older cached render survived prune")
	}
}

func TestFetchLogsAndCleanupSettings(t *testing.T) {
	repo := newTestRepo(t)
	feed, err := repo.CreateFeed(Feed{Title: "F", FeedURL: "https://example.com/rss"})
	if err != nil {
		t.Fatal(err)
	}

	// Record a success then a failure; both become rows, newest first.
	if err := repo.RecordFetchResult(feed.ID, `"v1"`, "", "", true); err != nil {
		t.Fatal(err)
	}
	if err := repo.RecordFetchResult(feed.ID, "", "", "upstream returned 500", false); err != nil {
		t.Fatal(err)
	}
	logs, err := repo.ListFetchLogs(0)
	if err != nil {
		t.Fatal(err)
	}
	if len(logs) != 2 {
		t.Fatalf("fetch logs = %+v, want 2 rows", logs)
	}
	if logs[0].Success || logs[0].Error != "upstream returned 500" || logs[0].FeedTitle != "F" {
		t.Fatalf("newest log = %+v", logs[0])
	}
	if !logs[1].Success || logs[1].Error != "" {
		t.Fatalf("older log = %+v", logs[1])
	}

	pruned, err := repo.PruneFetchLogs(time.Now().Add(24 * time.Hour)) // past all rows
	if err != nil {
		t.Fatal(err)
	}
	if pruned != 2 {
		t.Fatalf("pruned = %d, want 2", pruned)
	}
	if logs, _ := repo.ListFetchLogs(0); len(logs) != 0 {
		t.Fatalf("logs after prune = %+v, want empty", logs)
	}

	s, err := repo.GetSettings()
	if err != nil {
		t.Fatal(err)
	}
	if s.FetchLogCleanupSeconds != 30*86400 {
		t.Fatalf("default cleanup seconds = %d, want %d", s.FetchLogCleanupSeconds, 30*86400)
	}
	if err := repo.SetFetchLogCleanupSeconds(7 * 86400); err != nil {
		t.Fatal(err)
	}
	s, err = repo.GetSettings()
	if err != nil {
		t.Fatal(err)
	}
	if s.FetchLogCleanupSeconds != 7*86400 {
		t.Fatalf("cleanup seconds after set = %d, want %d", s.FetchLogCleanupSeconds, 7*86400)
	}
	if s.RenderCacheCleanupSeconds != 30*86400 {
		t.Fatalf("default render cache cleanup seconds = %d, want %d", s.RenderCacheCleanupSeconds, 30*86400)
	}
	if err := repo.SetRenderCacheCleanupSeconds(14 * 86400); err != nil {
		t.Fatal(err)
	}
	s, err = repo.GetSettings()
	if err != nil {
		t.Fatal(err)
	}
	if s.RenderCacheCleanupSeconds != 14*86400 {
		t.Fatalf("render cache cleanup seconds after set = %d, want %d", s.RenderCacheCleanupSeconds, 14*86400)
	}
	if s.FetchLogCleanupSeconds != 7*86400 {
		t.Fatalf("fetch log cleanup seconds regressed to %d after render set", s.FetchLogCleanupSeconds)
	}
}

// TestSetReadDoesNotRewriteFTS pins migration 0006: read/star toggles must not
// churn the FTS index, while content updates still do.
func TestSetReadDoesNotRewriteFTS(t *testing.T) {
	repo := newTestRepo(t)
	feed, err := repo.CreateFeed(Feed{Title: "F", FeedURL: "https://f.example/rss"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := repo.AddItems(feed.ID, []Item{{GUID: "g1", Title: "hello", Summary: "world", Content: "body"}}); err != nil {
		t.Fatal(err)
	}
	items, _, err := repo.ListItems(ItemFilter{})
	if err != nil || len(items) != 1 {
		t.Fatalf("list: err=%v n=%d", err, len(items))
	}
	if err := repo.SetRead(items[0].ID, true); err != nil {
		t.Fatal(err)
	}

	var ddl string
	if err := repo.DB.QueryRow(`SELECT sql FROM sqlite_master WHERE type='trigger' AND name='items_au'`).Scan(&ddl); err != nil {
		t.Fatalf("trigger ddl: %v", err)
	}
	if !strings.Contains(ddl, "UPDATE OF title, summary, content") {
		t.Fatalf("items_au not scoped to content columns: %s", ddl)
	}

	var found int
	if err := repo.DB.QueryRow(`SELECT COUNT(*) FROM items_fts WHERE items_fts MATCH 'hello'`).Scan(&found); err != nil {
		t.Fatalf("fts search: %v", err)
	}
	if found != 1 {
		t.Fatalf("fts rows after read toggle = %d, want 1", found)
	}
}

func TestRenderCache(t *testing.T) {
	repo := newTestRepo(t)

	if _, ok, err := repo.GetRenderCache("https://example.com/a"); err != nil || ok {
		t.Fatalf("unexpected cache hit: ok=%v err=%v", ok, err)
	}

	if err := repo.PutRenderCache("https://example.com/a", []byte("<p>one</p>")); err != nil {
		t.Fatal(err)
	}
	if got, ok, err := repo.GetRenderCache("https://example.com/a"); err != nil || !ok || string(got) != "<p>one</p>" {
		t.Fatalf("get after put = %q ok=%v err=%v", got, ok, err)
	}
	if err := repo.PutRenderCache("https://example.com/a", []byte("<p>two</p>")); err != nil {
		t.Fatal(err)
	}
	if got, _, _ := repo.GetRenderCache("https://example.com/a"); string(got) != "<p>two</p>" {
		t.Fatalf("get after repopulate = %q, want <p>two</p>", got)
	}

	if err := repo.PutRenderCache("https://example.com/b", []byte("<p>new</p>")); err != nil {
		t.Fatal(err)
	}
	pruned, err := repo.PruneRenderCache(time.Now().Add(24 * time.Hour)) // past all rows
	if err != nil {
		t.Fatal(err)
	}
	if pruned != 2 {
		t.Fatalf("pruned = %d, want 2", pruned)
	}
	if _, ok, _ := repo.GetRenderCache("https://example.com/a"); ok {
		t.Fatal("rendered article still cached after prune")
	}
}

// TestRefreshIntervalSetting pins the refresh_interval_seconds and
// min_refresh_gap_seconds settings round-trip (default from migration, then
// persisted and re-read).
func TestRefreshIntervalSetting(t *testing.T) {
	repo := newTestRepo(t)

	s, err := repo.GetSettings()
	if err != nil {
		t.Fatal(err)
	}
	if s.RefreshIntervalSeconds != DefaultRefreshIntervalSeconds {
		t.Fatalf("default refresh interval = %d, want %d",
			s.RefreshIntervalSeconds, DefaultRefreshIntervalSeconds)
	}
	if s.MinRefreshGapSeconds != DefaultMinRefreshGapSeconds {
		t.Fatalf("default min refresh gap = %d, want %d",
			s.MinRefreshGapSeconds, DefaultMinRefreshGapSeconds)
	}

	if err := repo.SetRefreshIntervalSeconds(1800); err != nil {
		t.Fatal(err)
	}
	if err := repo.SetMinRefreshGapSeconds(300); err != nil {
		t.Fatal(err)
	}
	s, err = repo.GetSettings()
	if err != nil {
		t.Fatal(err)
	}
	if s.RefreshIntervalSeconds != 1800 {
		t.Fatalf("refresh interval after set = %d, want 1800", s.RefreshIntervalSeconds)
	}
	if s.MinRefreshGapSeconds != 300 {
		t.Fatalf("min refresh gap after set = %d, want 300", s.MinRefreshGapSeconds)
	}
}
