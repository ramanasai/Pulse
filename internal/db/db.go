package db

import (
	"database/sql"
	"embed"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	_ "modernc.org/sqlite"
)

//go:embed schema.sql
var schemaFS embed.FS

func appDataDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	base := filepath.Join(home, ".local", "share", "pulse")
	if err := os.MkdirAll(base, 0o755); err != nil {
		return "", err
	}
	return base, nil
}

func Open() (*sql.DB, error) {
	dir, err := appDataDir()
	if err != nil {
		return nil, err
	}

	path := filepath.Join(dir, "pulse.db")
	dsn := fmt.Sprintf(
		"file:%s?_pragma=busy_timeout(5000)&_pragma=foreign_keys(ON)&_pragma=journal_mode(WAL)&_pragma=synchronous(NORMAL)",
		path,
	)

	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, err
	}

	if _, _ = db.Exec(`PRAGMA foreign_keys=ON; PRAGMA busy_timeout=5000; PRAGMA journal_mode=WAL; PRAGMA synchronous=NORMAL;`); false {
		// no-op; keep linter quiet
	}

	if err := migrate(db); err != nil {
		_ = db.Close()
		return nil, err
	}

	if err := EnsureThreadColumns(db); err != nil {
		_ = db.Close()
		return nil, err
	}

	if err := EnsureEncryptedColumn(db); err != nil {
		_ = db.Close()
		return nil, err
	}

	return db, nil
}

func migrate(db *sql.DB) error {
	b, err := schemaFS.ReadFile("schema.sql")
	if err != nil {
		return err
	}
	if _, err := db.Exec(string(b)); err != nil {
		return errors.Join(fmt.Errorf("schema apply failed"), err)
	}
	return nil
}

// ------------------------------
// Threads (idempotent upgrader)
// ------------------------------

func EnsureThreadColumns(db *sql.DB) error {
	needThread := true
	needParent := true

	rows, err := db.Query(`PRAGMA table_info(entries)`)
	if err != nil {
		return err
	}
	defer rows.Close()

	for rows.Next() {
		var cid int
		var name, ctype string
		var notnull, pk int
		var dflt sql.NullString
		if err := rows.Scan(&cid, &name, &ctype, &notnull, &dflt, &pk); err != nil {
			return err
		}
		switch strings.ToLower(name) {
		case "thread_id":
			needThread = false
		case "parent_id":
			needParent = false
		}
	}
	if err := rows.Err(); err != nil {
		return err
	}

	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()

	if needThread {
		if _, err := tx.Exec(`ALTER TABLE entries ADD COLUMN thread_id INTEGER`); err != nil {
			return fmt.Errorf("add thread_id: %w", err)
		}
	}
	if needParent {
		if _, err := tx.Exec(`ALTER TABLE entries ADD COLUMN parent_id INTEGER`); err != nil {
			return fmt.Errorf("add parent_id: %w", err)
		}
	}
	if _, err := tx.Exec(`CREATE INDEX IF NOT EXISTS idx_entries_thread ON entries(thread_id)`); err != nil {
		return err
	}
	if _, err := tx.Exec(`CREATE INDEX IF NOT EXISTS idx_entries_parent ON entries(parent_id)`); err != nil {
		return err
	}
	return tx.Commit()
}

// ------------------------------
// Migration Helper Functions
// ------------------------------
