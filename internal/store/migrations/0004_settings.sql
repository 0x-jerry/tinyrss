CREATE TABLE settings (
    key TEXT PRIMARY KEY,
    value TEXT NOT NULL DEFAULT ''
);

INSERT INTO settings(key, value) VALUES ('fetch_log_cleanup_days', '30');
