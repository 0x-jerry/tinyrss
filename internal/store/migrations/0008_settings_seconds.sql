INSERT INTO settings(key, value)
SELECT 'fetch_log_cleanup_seconds', printf('%d', CAST(value AS INTEGER) * 86400)
FROM settings WHERE key = 'fetch_log_cleanup_days';

INSERT INTO settings(key, value)
SELECT 'render_cache_cleanup_seconds', printf('%d', CAST(value AS INTEGER) * 86400)
FROM settings WHERE key = 'render_cache_cleanup_days';

INSERT INTO settings(key, value)
SELECT 'refresh_interval_seconds', printf('%d', CAST(value AS INTEGER) * 60)
FROM settings WHERE key = 'refresh_interval_minutes';

INSERT INTO settings(key, value) VALUES ('fetch_log_cleanup_seconds', '2592000') ON CONFLICT(key) DO NOTHING;
INSERT INTO settings(key, value) VALUES ('render_cache_cleanup_seconds', '2592000') ON CONFLICT(key) DO NOTHING;
INSERT INTO settings(key, value) VALUES ('refresh_interval_seconds', '900') ON CONFLICT(key) DO NOTHING;
INSERT INTO settings(key, value) VALUES ('min_refresh_gap_seconds', '600');

DELETE FROM settings WHERE key IN
  ('fetch_log_cleanup_days','render_cache_cleanup_days','refresh_interval_minutes');
