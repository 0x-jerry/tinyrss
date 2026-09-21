package repository

import (
	"reflect"
	"testing"
)

func TestGetFeedStats(t *testing.T) {
	repo := newTestRepo(t)
	a, err := repo.CreateFeed(Feed{Title: "A", FeedURL: "https://a.example/rss"})
	if err != nil {
		t.Fatal(err)
	}
	b, err := repo.CreateFeed(Feed{Title: "B", FeedURL: "https://b.example/rss"})
	if err != nil {
		t.Fatal(err)
	}
	c, err := repo.CreateFeed(Feed{Title: "C", FeedURL: "https://c.example/rss"})
	if err != nil {
		t.Fatal(err)
	}

	if n, err := repo.AddItems(a.ID, []Item{
		{GUID: "a1", Title: "t", PublishedAt: "2024-05-01 10:00:00"},
		{GUID: "a2", Title: "t", PublishedAt: "2024-05-01 12:00:00"},
		{GUID: "a3", Title: "t", PublishedAt: "2024-05-02 09:00:00"},
	}); err != nil || n != 3 {
		t.Fatalf("add A items: n=%d err=%v", n, err)
	}
	if n, err := repo.AddItems(b.ID, []Item{
		{GUID: "b1", Title: "t", PublishedAt: "2024-06-10 08:00:00"},
	}); err != nil || n != 1 {
		t.Fatalf("add B items: n=%d err=%v", n, err)
	}

	stats, err := repo.GetFeedStats(30)
	if err != nil {
		t.Fatalf("GetFeedStats: %v", err)
	}
	byID := map[int]FeedStat{}
	for _, s := range stats {
		byID[s.FeedID] = s
	}

	aStat, ok := byID[a.ID]
	if !ok {
		t.Fatalf("feed A missing from stats: %+v", stats)
	}
	if aStat.Total != 3 {
		t.Fatalf("A total = %d, want 3", aStat.Total)
	}
	if aStat.LatestAt != "2024-05-02 09:00:00" {
		t.Fatalf("A latest = %q", aStat.LatestAt)
	}
	wantSeries := []DayCount{{Date: "2024-05-01", Count: 2}, {Date: "2024-05-02", Count: 1}}
	if !reflect.DeepEqual(aStat.Series, wantSeries) {
		t.Fatalf("A series = %+v, want %+v", aStat.Series, wantSeries)
	}

	bStat, ok := byID[b.ID]
	if !ok {
		t.Fatalf("feed B missing from stats")
	}
	if bStat.Total != 1 {
		t.Fatalf("B total = %d, want 1", bStat.Total)
	}
	if bStat.LatestAt != "2024-06-10 08:00:00" {
		t.Fatalf("B latest = %q", bStat.LatestAt)
	}

	cStat, ok := byID[c.ID]
	if !ok {
		t.Fatalf("feed C missing from stats")
	}
	if cStat.Total != 0 {
		t.Fatalf("C total = %d, want 0", cStat.Total)
	}
	if cStat.LatestAt != "" {
		t.Fatalf("C latest = %q, want empty", cStat.LatestAt)
	}
	if len(cStat.Series) != 0 {
		t.Fatalf("C series = %+v, want empty", cStat.Series)
	}
}
