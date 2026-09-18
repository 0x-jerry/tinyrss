-- Read/star toggles must not rewrite the FTS index; only content changes do.
DROP TRIGGER items_au;

CREATE TRIGGER items_au AFTER UPDATE OF title, summary, content ON items BEGIN
    DELETE FROM items_fts WHERE rowid = old.id;
    INSERT INTO items_fts(rowid, title, summary, content)
    VALUES (new.id, new.title, new.summary, new.content);
END;

CREATE INDEX idx_items_feed_published ON items(feed_id, published_at DESC);
CREATE INDEX idx_items_unread ON items(is_read) WHERE is_read = 0;
CREATE INDEX idx_items_starred ON items(is_starred) WHERE is_starred = 1;
