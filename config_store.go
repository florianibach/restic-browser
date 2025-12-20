package main

import (
	"context"
	"database/sql"
	"errors"
	"time"

	_ "modernc.org/sqlite"
)

type RepoConfig struct {
	ID        string
	Path      string
	Password  string
	NoLock    bool
	CreatedAt time.Time
	UpdatedAt time.Time
}

type ConfigStore struct {
	db *sql.DB
}

func OpenConfigStore(dbPath string) (*ConfigStore, error) {
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, err
	}
	// sensible defaults
	db.SetMaxOpenConns(1)

	s := &ConfigStore{db: db}
	if err := s.migrate(); err != nil {
		_ = db.Close()
		return nil, err
	}
	return s, nil
}

func (s *ConfigStore) Close() error { return s.db.Close() }

func (s *ConfigStore) migrate() error {
	_, err := s.db.Exec(`
CREATE TABLE IF NOT EXISTS repositories (
  id TEXT PRIMARY KEY,
  path TEXT NOT NULL,
  password TEXT NOT NULL,
  no_lock INTEGER NOT NULL DEFAULT 1,
  created_at TEXT NOT NULL,
  updated_at TEXT NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_repositories_path ON repositories(path);
`)
	return err
}

func (s *ConfigStore) GetRepo(ctx context.Context, id string) (RepoConfig, bool, error) {
	var r RepoConfig
	var noLock int
	var created, updated string

	err := s.db.QueryRowContext(ctx,
		`SELECT id, path, password, no_lock, created_at, updated_at FROM repositories WHERE id = ?`,
		id,
	).Scan(&r.ID, &r.Path, &r.Password, &noLock, &created, &updated)

	if errors.Is(err, sql.ErrNoRows) {
		return RepoConfig{}, false, nil
	}
	if err != nil {
		return RepoConfig{}, false, err
	}

	r.NoLock = noLock != 0
	r.CreatedAt, _ = time.Parse(time.RFC3339, created)
	r.UpdatedAt, _ = time.Parse(time.RFC3339, updated)
	return r, true, nil
}

func (s *ConfigStore) Upsert(ctx context.Context, r RepoConfig) error {
	now := time.Now().UTC().Format(time.RFC3339)
	noLock := 0
	if r.NoLock {
		noLock = 1
	}

	_, err := s.db.ExecContext(ctx, `
INSERT INTO repositories (id, path, password, no_lock, created_at, updated_at)
VALUES (?, ?, ?, ?, ?, ?)
ON CONFLICT(id) DO UPDATE SET
  path = excluded.path,
  password = excluded.password,
  no_lock = excluded.no_lock,
  updated_at = excluded.updated_at
`, r.ID, r.Path, r.Password, noLock, now, now)

	return err
}

func (s *ConfigStore) List(ctx context.Context) ([]RepoConfig, error) {
	rows, err := s.db.QueryContext(ctx, `
SELECT id, path, password, no_lock, created_at, updated_at
FROM repositories
ORDER BY id ASC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []RepoConfig
	for rows.Next() {
		var r RepoConfig
		var noLock int
		var created, updated string
		if err := rows.Scan(&r.ID, &r.Path, &r.Password, &noLock, &created, &updated); err != nil {
			return nil, err
		}
		r.NoLock = noLock != 0
		r.CreatedAt, _ = time.Parse(time.RFC3339, created)
		r.UpdatedAt, _ = time.Parse(time.RFC3339, updated)
		out = append(out, r)
	}
	return out, rows.Err()
}
