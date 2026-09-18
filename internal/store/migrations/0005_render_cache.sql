CREATE TABLE render_cache (
    url TEXT PRIMARY KEY,
    html TEXT NOT NULL,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

INSERT INTO settings(key, value) VALUES ('render_cache_cleanup_days', '30');
