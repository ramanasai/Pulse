PRAGMA journal_mode=WAL;
PRAGMA foreign_keys=ON;


CREATE TABLE IF NOT EXISTS entries (
id INTEGER PRIMARY KEY,
ts DATETIME NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ','now')),
category TEXT NOT NULL DEFAULT 'note',
text TEXT,
project TEXT,
tags TEXT, -- comma separated for simplicity
duration_minutes INTEGER DEFAULT 0,
encrypted BOOLEAN DEFAULT FALSE,
thread_id INTEGER,
parent_id INTEGER,
FOREIGN KEY (thread_id) REFERENCES entries(id) ON DELETE SET NULL,
FOREIGN KEY (parent_id) REFERENCES entries(id) ON DELETE SET NULL
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

-- Composite indexes for common query patterns
CREATE INDEX IF NOT EXISTS idx_entries_project_ts ON entries(project, ts);
CREATE INDEX IF NOT EXISTS idx_entries_category_ts ON entries(category, ts);
CREATE INDEX IF NOT EXISTS idx_entries_thread_parent ON entries(thread_id, parent_id);

-- Templates table
CREATE TABLE IF NOT EXISTS templates (
id TEXT PRIMARY KEY,
name TEXT NOT NULL,
category TEXT NOT NULL,
content TEXT NOT NULL,
description TEXT,
variables TEXT, -- JSON array of variable names
is_custom BOOLEAN DEFAULT FALSE,
usage_count INTEGER DEFAULT 0,
last_used DATETIME,
is_favorite BOOLEAN DEFAULT FALSE,
created_at DATETIME DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ','now')),
updated_at DATETIME DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ','now'))
);

CREATE INDEX IF NOT EXISTS idx_templates_category ON templates(category);
CREATE INDEX IF NOT EXISTS idx_templates_favorite ON templates(is_favorite);