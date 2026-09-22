package repository

import (
	"context"
	"database/sql"
	"strings"
)

// AddItems upserts parsed items keyed by (feed_id, guid). Newly inserted rows
// default to unread; existing rows keep their read/star state while their
// metadata is refreshed. Returns the count of new (unread) items.
func (r *Repo) AddItems(feedID int, items []Item) (int, error) {
	if len(items) == 0 {
		return 0, nil
	}
	tx, err := r.DB.BeginTx(context.Background(), nil)
	if err != nil {
		return 0, err
	}
	defer tx.Rollback()

	ins, err := tx.Prepare(`INSERT OR IGNORE INTO items
		(feed_id, guid, title, url, author, summary, content, published_at, is_read, is_starred)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, 0, 0)`)
	if err != nil {
		return 0, err
	}
	defer ins.Close()
	upd, err := tx.Prepare(`UPDATE items SET title = ?, url = ?, author = ?,
		summary = ?, content = ?, published_at = ? WHERE feed_id = ? AND guid = ?`)
	if err != nil {
		return 0, err
	}
	defer upd.Close()

	newCount := 0
	for _, it := range items {
		res, err := ins.Exec(feedID, it.GUID, it.Title, it.URL, it.Author,
			it.Summary, it.Content, nullTime(it.PublishedAt))
		if err != nil {
			return 0, err
		}
		if n, _ := res.RowsAffected(); n > 0 {
			newCount++
			continue
		}
		if _, err := upd.Exec(it.Title, it.URL, it.Author, it.Summary, it.Content,
			nullTime(it.PublishedAt), feedID, it.GUID); err != nil {
			return 0, err
		}
	}
	return newCount, tx.Commit()
}

func nullTime(s string) any {
	if s == "" {
		return nil
	}
	return s
}

// ListItems returns the paginated, filtered items plus the total matching count.
func (r *Repo) ListItems(filter ItemFilter) ([]Item, int, error) {
	page := filter.Page
	if page < 1 {
		page = 1
	}
	limit := filter.Limit
	if limit <= 0 {
		limit = 50
	}
	if limit > 500 {
		limit = 500
	}
	search := ftsQuery(filter.Search)

	where, args := itemWhere(filter, search)
	countQuery := `SELECT COUNT(*) FROM items it JOIN feeds f ON f.id = it.feed_id WHERE ` + where
	var total int
	if err := r.DB.QueryRow(countQuery, args...).Scan(&total); err != nil {
		return nil, 0, err
	}

	listQuery := `SELECT it.id, it.feed_id, f.title AS feed_title, it.title, it.url, it.author,
		it.published_at, it.is_read, it.is_starred, it.created_at
		FROM items it JOIN feeds f ON f.id = it.feed_id WHERE ` + where +
		` ORDER BY it.published_at IS NULL, it.published_at DESC, it.id DESC LIMIT ? OFFSET ?`
	args = append(args, limit, (page-1)*limit)

	rows, err := r.DB.Query(listQuery, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	out := []Item{}
	for rows.Next() {
		var it Item
		var author, pub, created sql.NullString
		if err := rows.Scan(&it.ID, &it.FeedID, &it.FeedTitle, &it.Title, &it.URL, &author,
			&pub, &it.IsRead, &it.IsStarred, &created); err != nil {
			return nil, 0, err
		}
		if author.Valid {
			it.Author = author.String
		}
		if pub.Valid {
			it.PublishedAt = pub.String
		}
		if created.Valid {
			it.CreatedAt = created.String
		}
		out = append(out, it)
	}
	return out, total, rows.Err()
}

// itemWhere builds the WHERE clause (no leading "WHERE") and bound args.
func itemWhere(filter ItemFilter, search string) (string, []any) {
	conds := []string{"1=1"}
	args := []any{}
	if filter.FeedID > 0 {
		conds = append(conds, "it.feed_id = ?")
		args = append(args, filter.FeedID)
	}
	if filter.Unread {
		conds = append(conds, "it.is_read = 0")
	}
	if filter.Starred {
		conds = append(conds, "it.is_starred = 1")
	}
	if search != "" {
		conds = append(conds, "it.id IN (SELECT rowid FROM items_fts WHERE items_fts MATCH ?)")
		args = append(args, search)
	}
	return strings.Join(conds, " AND "), args
}

// ftsQuery escapes a user query for an FTS5 MATCH string. Each whitespace
// token is quoted (quotes doubled) and ANDed, so special FTS syntax is treated
// literally. Empty input returns "" meaning "no search filter".
func ftsQuery(q string) string {
	q = strings.TrimSpace(q)
	if q == "" {
		return ""
	}
	fields := strings.Fields(q)
	parts := make([]string, 0, len(fields))
	for _, f := range fields {
		parts = append(parts, `"`+strings.ReplaceAll(f, `"`, `""`)+`"`)
	}
	return strings.Join(parts, " ")
}

func (r *Repo) GetItem(id int) (Item, error) {
	var it Item
	var author, pub, created sql.NullString
	err := r.DB.QueryRow(`SELECT it.id, it.feed_id, f.title, it.title, it.url, it.author,
		it.summary, it.content, it.published_at, it.is_read, it.is_starred, it.created_at
		FROM items it JOIN feeds f ON f.id = it.feed_id WHERE it.id = ?`, id).
		Scan(&it.ID, &it.FeedID, &it.FeedTitle, &it.Title, &it.URL, &author,
			&it.Summary, &it.Content, &pub, &it.IsRead, &it.IsStarred, &created)
	if err == sql.ErrNoRows {
		return it, ErrNotFound{what: "item"}
	}
	if author.Valid {
		it.Author = author.String
	}
	if pub.Valid {
		it.PublishedAt = pub.String
	}
	if created.Valid {
		it.CreatedAt = created.String
	}
	return it, err
}

func (r *Repo) SetRead(id int, read bool) error {
	res, err := r.DB.Exec(`UPDATE items SET is_read = ? WHERE id = ?`, boolInt(read), id)
	if err != nil {
		return err
	}
	return requireAffected(res, "item")
}

func (r *Repo) SetStarred(id int, starred bool) error {
	res, err := r.DB.Exec(`UPDATE items SET is_starred = ? WHERE id = ?`, boolInt(starred), id)
	if err != nil {
		return err
	}
	return requireAffected(res, "item")
}

// MarkAllRead marks every item matching the filters (empty filter = all) as read.
func (r *Repo) MarkAllRead(filter ItemFilter) (int, error) {
	where, args := itemWhere(filter, "")
	query := `UPDATE items SET is_read = 1 WHERE id IN (SELECT it.id FROM items it
		JOIN feeds f ON f.id = it.feed_id WHERE ` + where + `)`
	res, err := r.DB.Exec(query, args...)
	if err != nil {
		return 0, err
	}
	n, _ := res.RowsAffected()
	return int(n), nil
}

func (r *Repo) Counts() (feeds, items, unread int, err error) {
	if err = r.DB.QueryRow(`SELECT COUNT(*) FROM feeds`).Scan(&feeds); err != nil {
		return
	}
	if err = r.DB.QueryRow(`SELECT COUNT(*) FROM items`).Scan(&items); err != nil {
		return
	}
	if err = r.DB.QueryRow(`SELECT COUNT(*) FROM items WHERE is_read = 0`).Scan(&unread); err != nil {
		return
	}
	return
}
