// Package store owns the raw SQLite connection: opening, PRAGMAs and
// versioned schema migrations. Domain logic lives in internal/feeds.
package store

import (
	"database/sql"
	"embed"
	"fmt"
	"io/fs"
	"path/filepath"
	"sort"
	"strings"

	_ "modernc.org/sqlite"
)

//go:embed migrations/*.sql
var migrationsFS embed.FS

type Store struct {
	DB *sql.DB
}

// Open opens (or creates) the SQLite database at path, applies PRAGMAs and
// pending migrations, and returns a ready Store. Use ":memory:" for tests.
func Open(path string) (*Store, error) {
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, err
	}
	// One shared connection serializes writes; busy_timeout covers transient
	// contention. WAL keeps readers going while a single writer holds it.
	db.SetMaxOpenConns(1)
	for _, pragma := range []string{
		"PRAGMA journal_mode=WAL;",
		"PRAGMA foreign_keys=ON;",
		"PRAGMA busy_timeout=5000;",
		"PRAGMA synchronous=NORMAL;",
	} {
		if _, err := db.Exec(pragma); err != nil {
			db.Close()
			return nil, fmt.Errorf("apply pragma %q: %w", pragma, err)
		}
	}
	s := &Store{DB: db}
	if err := s.migrate(); err != nil {
		db.Close()
		return nil, err
	}
	return s, nil
}

func (s *Store) Close() error {
	return s.DB.Close()
}

func (s *Store) migrate() error {
	if _, err := s.DB.Exec(`CREATE TABLE IF NOT EXISTS schema_migrations(
		version TEXT PRIMARY KEY,
		applied_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
	)`); err != nil {
		return fmt.Errorf("create schema_migrations: %w", err)
	}
	entries, err := fs.Glob(migrationsFS, "migrations/*.sql")
	if err != nil {
		return err
	}
	sort.Strings(entries)
	for _, name := range entries {
		version := strings.TrimSuffix(filepath.Base(name), ".sql")
		var applied int
		if err := s.DB.QueryRow(
			`SELECT COUNT(*) FROM schema_migrations WHERE version = ?`, version,
		).Scan(&applied); err != nil {
			return err
		}
		if applied > 0 {
			continue
		}
		sqlBytes, err := migrationsFS.ReadFile(name)
		if err != nil {
			return err
		}
		tx, err := s.DB.Begin()
		if err != nil {
			return err
		}
		if _, err := tx.Exec(string(sqlBytes)); err != nil {
			tx.Rollback()
			return fmt.Errorf("apply migration %s: %w", version, err)
		}
		if _, err := tx.Exec(
			`INSERT INTO schema_migrations(version) VALUES (?)`, version,
		); err != nil {
			tx.Rollback()
			return err
		}
		if err := tx.Commit(); err != nil {
			return err
		}
	}
	return nil
}
