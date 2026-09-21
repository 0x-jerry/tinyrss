package repository

import (
	"database/sql"
	"strconv"
)

// DayCount is one article-count bucket for a feed on a single day.
type DayCount struct {
	Date  string `json:"date"`
	Count int    `json:"count"`
}

// FeedStat summarizes one feed over a rolling window: total in-window articles,
// the latest article time, and the per-day count series.
type FeedStat struct {
	FeedID   int        `json:"feed_id"`
	Title    string     `json:"title"`
	LatestAt string     `json:"latest_at"`
	Total    int        `json:"total"`
	Series   []DayCount `json:"series"`
}

// GetFeedStats returns one FeedStat per feed over the trailing `days`.
// Articles are bucketed by published date, falling back to creation date.
// Feeds with no in-window articles still appear with an empty series.
func (r *Repo) GetFeedStats(days int) ([]FeedStat, error) {
	window := "-" + strconv.Itoa(days) + " days"

	feeds, err := r.ListFeeds()
	if err != nil {
		return nil, err
	}
	byID := make(map[int]int, len(feeds))
	stats := make([]FeedStat, len(feeds))
	for i, f := range feeds {
		byID[f.ID] = i
		stats[i] = FeedStat{FeedID: f.ID, Title: f.Title, Series: []DayCount{}}
	}

	rows, err := r.DB.Query(`SELECT it.feed_id, COUNT(*),
		MAX(COALESCE(NULLIF(it.published_at, ''), it.created_at))
		FROM items it JOIN feeds f ON f.id = it.feed_id
		WHERE it.created_at >= datetime('now', ?)
		GROUP BY it.feed_id`, window)
	if err != nil {
		return nil, err
	}
	for rows.Next() {
		var feedID, total int
		var latest sql.NullString
		if err := rows.Scan(&feedID, &total, &latest); err != nil {
			rows.Close()
			return nil, err
		}
		i, ok := byID[feedID]
		if !ok {
			continue
		}
		stats[i].Total = total
		if latest.Valid {
			stats[i].LatestAt = latest.String
		}
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return nil, err
	}
	rows.Close()

	srows, err := r.DB.Query(`SELECT it.feed_id,
		date(COALESCE(NULLIF(it.published_at, ''), it.created_at)) AS day, COUNT(*)
		FROM items it JOIN feeds f ON f.id = it.feed_id
		WHERE it.created_at >= datetime('now', ?)
		GROUP BY it.feed_id, day ORDER BY day`, window)
	if err != nil {
		return nil, err
	}
	for srows.Next() {
		var feedID, count int
		var day string
		if err := srows.Scan(&feedID, &day, &count); err != nil {
			srows.Close()
			return nil, err
		}
		i, ok := byID[feedID]
		if !ok {
			continue
		}
		stats[i].Series = append(stats[i].Series, DayCount{Date: day, Count: count})
	}
	if err := srows.Err(); err != nil {
		srows.Close()
		return nil, err
	}
	srows.Close()

	return stats, nil
}
