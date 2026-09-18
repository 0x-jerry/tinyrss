package feeds

import (
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
	if n != 4 {
		t.Fatalf("want 4 applied migrations, got %d", n)
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

	// First insert: both items are new (unread).
	first := []Item{
		{GUID: "a", Title: "One", URL: "https://example.com/1", Content: "c1"},
		{GUID: "b", Title: "Two", URL: "https://example.com/2", Content: "c2"},
	}
	if n, err := repo.AddItems(feed.ID, first); err != nil || n != 2 {
		t.Fatalf("first AddItems: n=%d err=%v", n, err)
	}
	// Re-inserting the same GUIDs adds nothing (dedup).
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

	// Mark read via ListItems + SetRead.
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

	// Folder-scoped mark-all-read touches both matching items.
	count, err := repo.MarkAllRead(ItemFilter{FolderID: folder.ID})
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

func TestFetchLogsAndCleanupSettings(t *testing.T) {
	repo := newTestRepo(t)
	feed, err := repo.CreateFeed(Feed{Title: "F", FeedURL: "https://example.com/rss"})
	if err != nil {
		t.Fatal(err)
	}

	// Record a success then a failure; both become rows, newest first.
	if err := repo.recordFetchResult(feed.ID, `"v1"`, "", "", true); err != nil {
		t.Fatal(err)
	}
	if err := repo.recordFetchResult(feed.ID, "", "", "upstream returned 500", false); err != nil {
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

	// Prune removes only rows older than the cutoff.
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

	// Cleanup retention defaults to 30 days, persists round-trip.
	s, err := repo.GetSettings()
	if err != nil {
		t.Fatal(err)
	}
	if s.FetchLogCleanupDays != 30 {
		t.Fatalf("default cleanup days = %d, want 30", s.FetchLogCleanupDays)
	}
	if err := repo.SetFetchLogCleanupDays(7); err != nil {
		t.Fatal(err)
	}
	s, err = repo.GetSettings()
	if err != nil {
		t.Fatal(err)
	}
	if s.FetchLogCleanupDays != 7 {
		t.Fatalf("cleanup days after set = %d, want 7", s.FetchLogCleanupDays)
	}
}
