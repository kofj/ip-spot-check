package quota

import (
	"context"
	"database/sql"
	"fmt"
	"time"
)

type exceededError struct{}

func (exceededError) Error() string { return "额度耗尽" }

var ErrExceeded error = exceededError{}

type Guard interface {
	Acquire(ctx context.Context, source string) error
	Usage(ctx context.Context, source string) (used, limit int)
}

type SQLGuard struct {
	db      *sql.DB
	limitOf func(source string) int
	now     func() time.Time
}

func NewSQLGuard(db *sql.DB, limitOf func(source string) int) *SQLGuard {
	return &SQLGuard{db: db, limitOf: limitOf, now: time.Now}
}

func (g *SQLGuard) Acquire(ctx context.Context, source string) error {
	limit := g.limitOf(source)
	if limit <= 0 {
		return fmt.Errorf("未配置限额: %s", source)
	}
	day := g.now().UTC().Format("2006-01-02")

	res, err := g.db.ExecContext(ctx,
		`UPDATE quota_usage SET used = used + 1 WHERE source = ? AND day = ? AND used < ?`,
		source, day, limit)
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n == 1 {
		return nil
	}

	_, err = g.db.ExecContext(ctx,
		`INSERT OR IGNORE INTO quota_usage (source, day, used) VALUES (?, ?, 0)`,
		source, day)
	if err != nil {
		return err
	}
	res, err = g.db.ExecContext(ctx,
		`UPDATE quota_usage SET used = used + 1 WHERE source = ? AND day = ? AND used < ?`,
		source, day, limit)
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n == 1 {
		return nil
	}
	return ErrExceeded
}

func (g *SQLGuard) Usage(ctx context.Context, source string) (used, limit int) {
	day := g.now().UTC().Format("2006-01-02")
	_ = g.db.QueryRowContext(ctx,
		`SELECT used FROM quota_usage WHERE source = ? AND day = ?`, source, day).Scan(&used)
	return used, g.limitOf(source)
}
