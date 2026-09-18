package repository

import (
	"database/sql"
	"strconv"
)

// settingValue reads one settings row, defaulting to def when the key is
// absent or unparsable.
func (r *Repo) settingValue(key string, def int) (int, error) {
	var v string
	switch err := r.DB.QueryRow(`SELECT value FROM settings WHERE key = ?`, key).Scan(&v); {
	case err == sql.ErrNoRows:
		return def, nil
	case err != nil:
		return def, err
	}
	if n, err := strconv.Atoi(v); err == nil {
		return n, nil
	}
	return def, nil
}

func (r *Repo) setSetting(key string, days int) error {
	_, err := r.DB.Exec(`INSERT INTO settings(key, value) VALUES (?, ?)
		ON CONFLICT(key) DO UPDATE SET value = excluded.value`, key, strconv.Itoa(days))
	return err
}

// GetSettings returns the auto-clean retentions, defaulting on (30 days) only
// when no value has ever been stored for a key.
func (r *Repo) GetSettings() (Settings, error) {
	fl, err := r.settingValue("fetch_log_cleanup_days", defaultFetchLogCleanupDays)
	if err != nil {
		return Settings{}, err
	}
	rc, err := r.settingValue("render_cache_cleanup_days", defaultRenderCacheCleanupDays)
	if err != nil {
		return Settings{}, err
	}
	return Settings{FetchLogCleanupDays: fl, RenderCacheCleanupDays: rc}, nil
}

// SetFetchLogCleanupDays stores the fetch-log auto-clean retention; 0 disables.
func (r *Repo) SetFetchLogCleanupDays(days int) error {
	return r.setSetting("fetch_log_cleanup_days", days)
}

// SetRenderCacheCleanupDays stores the render-cache auto-clean retention; 0 disables.
func (r *Repo) SetRenderCacheCleanupDays(days int) error {
	return r.setSetting("render_cache_cleanup_days", days)
}
