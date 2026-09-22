package repository

import (
	"database/sql"
	"encoding/json"
	"strings"
)

const feedCols = `id, title, feed_url, site_url, description, proxy_url, render_mode, folder_id,
	etag, last_modified, last_fetched_at, fetch_error, created_at, updated_at,
	(SELECT COUNT(*) FROM items i WHERE i.feed_id = feeds.id AND i.is_read = 0)`

func scanFeed(sc interface{ Scan(...any) error }) (Feed, error) {
	var f Feed
	var folderID sql.NullInt64
	var lastFetched sql.NullString
	err := sc.Scan(&f.ID, &f.Title, &f.FeedURL, &f.SiteURL, &f.Description, &f.ProxyURL, &f.RenderMode, &folderID,
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
	res, err := r.DB.Exec(`INSERT INTO feeds(title, feed_url, site_url, description, proxy_url, folder_id)
		VALUES (?, ?, ?, ?, ?, ?)`,
		f.Title, f.FeedURL, f.SiteURL, f.Description, f.ProxyURL, folderID)
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

// FeedRef is the scheduler's lightweight view of a feed: enough to decide due
// status and label progress, without the unread count and metadata ListFeeds
// loads for the API.
type FeedRef struct {
	ID            int
	Title         string
	LastFetchedAt string
}

func (r *Repo) ListFeedRefs() ([]FeedRef, error) {
	rows, err := r.DB.Query(`SELECT id, title, last_fetched_at FROM feeds`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []FeedRef{}
	for rows.Next() {
		var ref FeedRef
		var lastFetched sql.NullString
		if err := rows.Scan(&ref.ID, &ref.Title, &lastFetched); err != nil {
			return nil, err
		}
		if lastFetched.Valid {
			ref.LastFetchedAt = lastFetched.String
		}
		out = append(out, ref)
	}
	return out, rows.Err()
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

// FolderField is the folder_id intent of an update request. The zero value
// means the key was absent (leave the folder unchanged); Null=true means JSON
// null (clear it to Uncategorized); Set=true means reassign to ID.
type FolderField struct {
	Null bool
	Set  bool
	ID   int
}

func (f *FolderField) UnmarshalJSON(b []byte) error {
	if string(b) == "null" {
		f.Null = true
		f.Set = false
		return nil
	}
	f.Set = true
	return json.Unmarshal(b, &f.ID)
}

// UpdateFeed applies only the fields that were supplied; nil pointers (and an
// absent folder field) leave the corresponding column unchanged. A supplied
// folder_id is validated to reference an existing folder unless it is Null.
func (r *Repo) UpdateFeed(id int, title *string, feedURL, siteURL, description, proxyURL *string, folder FolderField) (Feed, error) {
	sets := []string{"updated_at = CURRENT_TIMESTAMP"}
	args := []any{}
	if title != nil {
		sets = append(sets, "title = ?")
		args = append(args, *title)
	}
	if feedURL != nil {
		sets = append(sets, "feed_url = ?")
		args = append(args, *feedURL)
	}
	if siteURL != nil {
		sets = append(sets, "site_url = ?")
		args = append(args, *siteURL)
	}
	if description != nil {
		sets = append(sets, "description = ?")
		args = append(args, *description)
	}
	if proxyURL != nil {
		sets = append(sets, "proxy_url = ?")
		args = append(args, *proxyURL)
	}
	if folder.Null || folder.Set {
		sets = append(sets, "folder_id = ?")
		if folder.Null {
			args = append(args, nil)
		} else {
			if ok, err := r.feedExistsFolder(&folder.ID); err != nil {
				return Feed{}, err
			} else if !ok {
				return Feed{}, ErrNotFound{what: "folder"}
			}
			args = append(args, folder.ID)
		}
	}
	if len(args) == 0 {
		// Nothing supplied to change: return current without touching the row.
		return r.GetFeed(id)
	}
	args = append(args, id)
	res, err := r.DB.Exec(`UPDATE feeds SET `+strings.Join(sets, ", ")+` WHERE id = ?`, args...)
	if isUnique(err) {
		return Feed{}, ErrConflict{what: "feed_url already exists"}
	}
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

func (r *Repo) SetRenderMode(id int, mode int) (Feed, error) {
	res, err := r.DB.Exec(`UPDATE feeds SET render_mode = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ?`, mode, id)
	if err != nil {
		return Feed{}, err
	}
	if err := requireAffected(res, "feed"); err != nil {
		return Feed{}, err
	}
	return r.GetFeed(id)
}

// RecordFetchResult updates a feed after a fetch attempt regardless of outcome
// so a failing feed stays alive and is retried next cycle, and appends a row
// to the fetch log so every attempt is auditable.
func (r *Repo) RecordFetchResult(id int, etag, lastModified, fetchErr string, success bool) error {
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
	if err != nil {
		return err
	}
	_, err = r.DB.Exec(`INSERT INTO fetch_logs(feed_id, success, error) VALUES (?, ?, ?)`,
		id, boolInt(success), fetchErr)
	return err
}
