package store

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"time"

	_ "modernc.org/sqlite"
)

type Store struct {
	db *sql.DB
}

func Open(driver, dsn string, maxOpenConns int) (*Store, error) {
	if driver != "sqlite" {
		return nil, fmt.Errorf("不支持的驱动: %s", driver)
	}
	if dir := dsnDir(dsn); dir != "" {
		if err := os.MkdirAll(dir, 0o700); err != nil {
			return nil, fmt.Errorf("创建数据目录 %s: %w", dir, err)
		}
	}

	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(maxOpenConns)
	if err := db.Ping(); err != nil {
		db.Close()
		return nil, err
	}
	return &Store{db: db}, nil
}

func (s *Store) DB() *sql.DB { return s.db }

func (s *Store) Close() error { return s.db.Close() }

func (s *Store) SaveQuery(ctx context.Context, ip string, at time.Time) error {
	_, err := s.db.ExecContext(ctx, `INSERT INTO queries (ip, queried_at) VALUES (?, ?)`, ip, at.Unix())
	return err
}

func (s *Store) QueryCount(ctx context.Context, ip string) (int, error) {
	var n int
	err := s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM queries WHERE ip = ?`, ip).Scan(&n)
	return n, err
}

func dsnDir(dsn string) string {
	p := dsn
	if i := len("file:"); len(p) > i && p[:i] == "file:" {
		p = p[i:]
	}
	for i := 0; i < len(p); i++ {
		if p[i] == '?' {
			p = p[:i]
			break
		}
	}
	if p == ":memory:" || p == "" || p[0] == ':' {
		return ""
	}
	dir := filepath.Dir(p)
	if dir == "." {
		return ""
	}
	return dir
}
