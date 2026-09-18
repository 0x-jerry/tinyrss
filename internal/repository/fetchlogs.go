package repository

import (
	"database/sql"
	"time"
)

// ListFetchLogs returns the most recent fetch log rows, joined with each feed's
// current title, newest first.
func (r *Repo) ListFetchLogs(limit int) ([]FetchLog, error) {
	if limit <= 0 {
		limit = 100
	}
	if limit > 500 {
		limit = 500
	}
	rows, err := r.DB.Query(`SELECT fl.id, fl.feed_id, f.title, fl.success, fl.error, fl.fetched_at
		FROM fetch_logs fl JOIN feeds f ON f.id = fl.feed_id
		ORDER BY fl.id DESC LIMIT ?`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []FetchLog{}
	for rows.Next() {
		var l FetchLog
		var fetchedAt sql.NullString
		if err := rows.Scan(&l.ID, &l.FeedID, &l.FeedTitle, &l.Success, &l.Error, &fetchedAt); err != nil {
			return nil, err
		}
		if fetchedAt.Valid {
			l.FetchedAt = fetchedAt.String
		}
		out = append(out, l)
	}
	return out, rows.Err()
}

// PruneFetchLogs deletes log rows older than the cutoff; returns rows removed.
// Stored timestamps are UTC (CURRENT_TIMESTAMP), so the cutoff is normalised to
// UTC before the comparison regardless of the caller's local timezone.
func (r *Repo) PruneFetchLogs(olderThan time.Time) (int64, error) {
	res, err := r.DB.Exec(`DELETE FROM fetch_logs WHERE fetched_at < ?`, olderThan.UTC().Format(TimeLayout))
	if err != nil {
		return 0, err
	}
	return res.RowsAffected()
}
