package feeds

import (
	"context"
	"database/sql"
	"strings"
)

type Repo struct {
	DB *sql.DB
}

func NewRepo(db *sql.DB) *Repo { return &Repo{DB: db} }

// isUnique reports whether err is a UNIQUE constraint violation.
func isUnique(err error) bool {
	return err != nil && strings.Contains(err.Error(), "UNIQUE")
}

// ---- folders ----

func (r *Repo) CreateFolder(name string) (Folder, error) {
	var f Folder
	err := r.DB.QueryRow(`INSERT INTO folders(name, sort_order) VALUES (?, COALESCE((SELECT MAX(sort_order)+1 FROM folders), 0)) RETURNING id, name, sort_order`, name).
		Scan(&f.ID, &f.Name, &f.SortOrder)
	if isUnique(err) {
		return f, ErrConflict{what: "folder name already exists"}
	}
	return f, err
}

func (r *Repo) ListFolders() ([]Folder, error) {
	rows, err := r.DB.Query(`SELECT id, name, sort_order FROM folders ORDER BY sort_order, name`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Folder{}
	for rows.Next() {
		var f Folder
		if err := rows.Scan(&f.ID, &f.Name, &f.SortOrder); err != nil {
			return nil, err
		}
		out = append(out, f)
	}
	return out, rows.Err()
}

func (r *Repo) GetFolder(id int) (Folder, error) {
	var f Folder
	err := r.DB.QueryRow(`SELECT id, name, sort_order FROM folders WHERE id = ?`, id).
		Scan(&f.ID, &f.Name, &f.SortOrder)
	if err == sql.ErrNoRows {
		return f, ErrNotFound{what: "folder"}
	}
	return f, err
}

func (r *Repo) UpdateFolder(id int, name string) error {
	res, err := r.DB.Exec(`UPDATE folders SET name = ? WHERE id = ?`, name, id)
	if isUnique(err) {
		return ErrConflict{what: "folder name already exists"}
	}
	if err != nil {
		return err
	}
	return requireAffected(res, "folder")
}

func (r *Repo) DeleteFolder(id int) error {
	// ON DELETE SET NULL folds its feeds into uncategorized automatically.
	res, err := r.DB.Exec(`DELETE FROM folders WHERE id = ?`, id)
	if err != nil {
		return err
	}
	return requireAffected(res, "folder")
}

// ---- feeds ----

const feedCols = `id, title, feed_url, site_url, description, folder_id,
	etag, last_modified, last_fetched_at, fetch_error, created_at, updated_at,
	(SELECT COUNT(*) FROM items i WHERE i.feed_id = feeds.id AND i.is_read = 0)`

func scanFeed(sc interface{ Scan(...any) error }) (Feed, error) {
	var f Feed
	var folderID sql.NullInt64
	var lastFetched sql.NullString
	err := sc.Scan(&f.ID, &f.Title, &f.FeedURL, &f.SiteURL, &f.Description, &folderID,
		&f.ETag, &f.LastModified, &lastFetched, &f.FetchError, &f.CreatedAt, &f.UpdatedAt, &f.Unread)
	if folderID.Valid {
		id := int(folderID.Int64)
		f.FolderID = &id
	}
	if lastFetched.Valid {
		f.LastFetchedAt = lastFetched.String
	}
	return f, err
}

// feedExistsFolder reports whether a folder id is present, for FK checks.
func (r *Repo) feedExistsFolder(id *int) (bool, error) {
	if id == nil {
		return true, nil
	}
	var n int
	err := r.DB.QueryRow(`SELECT COUNT(*) FROM folders WHERE id = ?`, *id).Scan(&n)
	return n > 0, err
}

func (r *Repo) CreateFeed(f Feed) (Feed, error) {
	var folderID any
	if f.FolderID != nil {
		ok, err := r.feedExistsFolder(f.FolderID)
		if err != nil {
			return f, err
		}
		if !ok {
			return f, ErrNotFound{what: "folder"}
		}
		folderID = *f.FolderID
	}
	res, err := r.DB.Exec(`INSERT INTO feeds(title, feed_url, site_url, description, folder_id)
		VALUES (?, ?, ?, ?, ?)`,
		f.Title, f.FeedURL, f.SiteURL, f.Description, folderID)
	if isUnique(err) {
		return f, ErrConflict{what: "feed_url already exists"}
	}
	if err != nil {
		return f, err
	}
	id, err := res.LastInsertId()
	if err != nil {
		return f, err
	}
	return r.GetFeed(int(id))
}

func (r *Repo) GetFeed(id int) (Feed, error) {
	f, err := scanFeed(r.DB.QueryRow(`SELECT `+feedCols+` FROM feeds WHERE id = ?`, id))
	if err == sql.ErrNoRows {
		return f, ErrNotFound{what: "feed"}
	}
	return f, err
}

func (r *Repo) ListFeeds() ([]Feed, error) {
	rows, err := r.DB.Query(`SELECT ` + feedCols + ` FROM feeds ORDER BY title`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Feed{}
	for rows.Next() {
		f, err := scanFeed(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, f)
	}
	return out, rows.Err()
}

func (r *Repo) UpdateFeed(id int, title string, folderID *int) (Feed, error) {
	if folderID != nil {
		ok, err := r.feedExistsFolder(folderID)
		if err != nil {
			return Feed{}, err
		}
		if !ok {
			return Feed{}, ErrNotFound{what: "folder"}
		}
	}
	var folderVal any
	if folderID != nil {
		folderVal = *folderID
	}
	res, err := r.DB.Exec(`UPDATE feeds SET title = ?, folder_id = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ?`,
		title, folderVal, id)
	if err != nil {
		return Feed{}, err
	}
	if err := requireAffected(res, "feed"); err != nil {
		return Feed{}, err
	}
	return r.GetFeed(id)
}

func (r *Repo) DeleteFeed(id int) error {
	res, err := r.DB.Exec(`DELETE FROM feeds WHERE id = ?`, id)
	if err != nil {
		return err
	}
	return requireAffected(res, "feed")
}

// recordFetchResult updates a feed after a fetch attempt regardless of outcome
// so a failing feed stays alive and is retried next cycle.
func (r *Repo) recordFetchResult(id int, etag, lastModified, fetchErr string, success bool) error {
	var err error
	if success {
		_, err = r.DB.Exec(`UPDATE feeds SET etag = ?, last_modified = ?,
			last_fetched_at = CURRENT_TIMESTAMP, fetch_error = '', updated_at = CURRENT_TIMESTAMP
			WHERE id = ?`, etag, lastModified, id)
	} else if fetchErr != "" {
		_, err = r.DB.Exec(`UPDATE feeds SET fetch_error = ?,
			last_fetched_at = CURRENT_TIMESTAMP, updated_at = CURRENT_TIMESTAMP WHERE id = ?`,
			fetchErr, id)
	}
	return err
}

// ---- items ----

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
	if filter.FolderID > 0 {
		conds = append(conds, "f.folder_id = ?")
		args = append(args, filter.FolderID)
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

func requireAffected(res sql.Result, what string) error {
	n, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if n == 0 {
		return ErrNotFound{what: what}
	}
	return nil
}

func boolInt(b bool) int {
	if b {
		return 1
	}
	return 0
}
