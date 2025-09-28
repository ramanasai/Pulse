package db

import (
	"database/sql"
	"embed"
	"errors"
	"fmt"
	"os"
	"path/filepath"

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
	dsn := fmt.Sprintf("file:%s?_pragma=busy_timeout=5000&_pragma=foreign_keys=ON", path)
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, err
	}
	if err := migrate(db); err != nil {
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
