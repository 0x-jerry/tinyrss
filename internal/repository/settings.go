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

func (r *Repo) setSetting(key string, seconds int) error {
	_, err := r.DB.Exec(`INSERT INTO settings(key, value) VALUES (?, ?)
		ON CONFLICT(key) DO UPDATE SET value = excluded.value`, key, strconv.Itoa(seconds))
	return err
}

// GetSettings returns the user-adjustable settings, defaulting only when no
// value has ever been stored for a key.
func (r *Repo) GetSettings() (Settings, error) {
	fl, err := r.settingValue("fetch_log_cleanup_seconds", defaultFetchLogCleanupSeconds)
	if err != nil {
		return Settings{}, err
	}
	rc, err := r.settingValue("render_cache_cleanup_seconds", defaultRenderCacheCleanupSeconds)
	if err != nil {
		return Settings{}, err
	}
	ri, err := r.settingValue("refresh_interval_seconds", DefaultRefreshIntervalSeconds)
	if err != nil {
		return Settings{}, err
	}
	gap, err := r.settingValue("min_refresh_gap_seconds", DefaultMinRefreshGapSeconds)
	if err != nil {
		return Settings{}, err
	}
	return Settings{
		FetchLogCleanupSeconds:    fl,
		RenderCacheCleanupSeconds: rc,
		RefreshIntervalSeconds:    ri,
		MinRefreshGapSeconds:      gap,
	}, nil
}

// SetFetchLogCleanupSeconds stores the fetch-log auto-clean retention in
// seconds; 0 disables.
func (r *Repo) SetFetchLogCleanupSeconds(seconds int) error {
	return r.setSetting("fetch_log_cleanup_seconds", seconds)
}

// SetRenderCacheCleanupSeconds stores the render-cache auto-clean retention in
// seconds; 0 disables.
func (r *Repo) SetRenderCacheCleanupSeconds(seconds int) error {
	return r.setSetting("render_cache_cleanup_seconds", seconds)
}

// SetRefreshIntervalSeconds stores the auto-refresh poll interval in seconds.
func (r *Repo) SetRefreshIntervalSeconds(seconds int) error {
	return r.setSetting("refresh_interval_seconds", seconds)
}

// SetMinRefreshGapSeconds stores the minimum gap between manual refresh-alls
// in seconds.
func (r *Repo) SetMinRefreshGapSeconds(seconds int) error {
	return r.setSetting("min_refresh_gap_seconds", seconds)
}
