PRAGMA journal_mode=WAL;
PRAGMA foreign_keys=ON;


CREATE TABLE IF NOT EXISTS entries (
id INTEGER PRIMARY KEY,
ts DATETIME NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ','now')),
category TEXT NOT NULL DEFAULT 'note',
text TEXT NOT NULL,
project TEXT,
tags TEXT, -- comma separated for simplicity
duration_minutes INTEGER DEFAULT 0
);


-- FTS for fast search on text + tags
CREATE VIRTUAL TABLE IF NOT EXISTS entries_fts USING fts5(
text, tags, content='entries', content_rowid='id'
);


-- Triggers to keep FTS in sync
CREATE TRIGGER IF NOT EXISTS entries_ai AFTER INSERT ON entries BEGIN
INSERT INTO entries_fts(rowid, text, tags) VALUES (new.id, new.text, new.tags);
END;
CREATE TRIGGER IF NOT EXISTS entries_ad AFTER DELETE ON entries BEGIN
INSERT INTO entries_fts(entries_fts, rowid, text, tags) VALUES('delete', old.id, old.text, old.tags);
END;
CREATE TRIGGER IF NOT EXISTS entries_au AFTER UPDATE ON entries BEGIN
INSERT INTO entries_fts(entries_fts, rowid, text, tags) VALUES('delete', old.id, old.text, old.tags);
INSERT INTO entries_fts(rowid, text, tags) VALUES (new.id, new.text, new.tags);
END;


CREATE INDEX IF NOT EXISTS idx_entries_ts ON entries(ts);
CREATE INDEX IF NOT EXISTS idx_entries_category ON entries(category);
CREATE INDEX IF NOT EXISTS idx_entries_project ON entries(project);