CREATE TABLE folders (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    name TEXT NOT NULL UNIQUE,
    sort_order INTEGER NOT NULL DEFAULT 0
);

CREATE TABLE feeds (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    title TEXT NOT NULL,
    feed_url TEXT NOT NULL UNIQUE,
    site_url TEXT NOT NULL DEFAULT '',
    description TEXT NOT NULL DEFAULT '',
    folder_id INTEGER REFERENCES folders(id) ON DELETE SET NULL,
    etag TEXT NOT NULL DEFAULT '',
    last_modified TEXT NOT NULL DEFAULT '',
    last_fetched_at DATETIME,
    fetch_error TEXT NOT NULL DEFAULT '',
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE items (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    feed_id INTEGER NOT NULL REFERENCES feeds(id) ON DELETE CASCADE,
    guid TEXT NOT NULL,
    title TEXT NOT NULL DEFAULT '',
    url TEXT NOT NULL DEFAULT '',
    author TEXT NOT NULL DEFAULT '',
    summary TEXT NOT NULL DEFAULT '',
    content TEXT NOT NULL DEFAULT '',
    published_at DATETIME,
    is_read INTEGER NOT NULL DEFAULT 0,
    is_starred INTEGER NOT NULL DEFAULT 0,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(feed_id, guid)
);

CREATE VIRTUAL TABLE items_fts USING fts5(title, summary, content);

CREATE TRIGGER items_ai AFTER INSERT ON items BEGIN
    INSERT INTO items_fts(rowid, title, summary, content)
    VALUES (new.id, new.title, new.summary, new.content);
END;

CREATE TRIGGER items_ad AFTER DELETE ON items BEGIN
    DELETE FROM items_fts WHERE rowid = old.id;
END;

CREATE TRIGGER items_au AFTER UPDATE ON items BEGIN
    DELETE FROM items_fts WHERE rowid = old.id;
    INSERT INTO items_fts(rowid, title, summary, content)
    VALUES (new.id, new.title, new.summary, new.content);
END;
