package repository

import (
	"database/sql"
	"time"
)

// GetRenderCache returns a previously cached server-render result for url. The
// second return is false when nothing is cached for that URL.
func (r *Repo) GetRenderCache(url string) ([]byte, bool, error) {
	var html []byte
	switch err := r.DB.QueryRow(`SELECT html FROM render_cache WHERE url = ?`, url).Scan(&html); {
	case err == sql.ErrNoRows:
		return nil, false, nil
	case err != nil:
		return nil, false, err
	}
	return html, true, nil
}

// PutRenderCache stores (or refreshes) a server-render result for url.
func (r *Repo) PutRenderCache(url string, html []byte) error {
	_, err := r.DB.Exec(`INSERT INTO render_cache(url, html, created_at) VALUES (?, ?, CURRENT_TIMESTAMP)
		ON CONFLICT(url) DO UPDATE SET html = excluded.html, created_at = CURRENT_TIMESTAMP`, url, html)
	return err
}

// PruneRenderCache deletes cached renders older than the cutoff; returns rows
// removed. The cutoff is normalised to UTC like PruneFetchLogs.
func (r *Repo) PruneRenderCache(olderThan time.Time) (int64, error) {
	res, err := r.DB.Exec(`DELETE FROM render_cache WHERE created_at < ?`, olderThan.UTC().Format(TimeLayout))
	if err != nil {
		return 0, err
	}
	return res.RowsAffected()
}
