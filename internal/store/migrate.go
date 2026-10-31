package store

import (
	"database/sql"
	"embed"
	"errors"
	"fmt"
	"os"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/sqlite"
	"github.com/golang-migrate/migrate/v4/source/iofs"
)

//go:embed migrations/*.sql
var embeddedMigrations embed.FS

// migrate 的 sqlite 驱动在 Close 时会关闭底层 *sql.DB，
// 因此迁移使用独立连接，避免影响业务连接池。
func withMigrator(dsn string, fn func(*migrate.Migrate) error) error {
	if dir := dsnDir(dsn); dir != "" {
		if err := os.MkdirAll(dir, 0o700); err != nil {
			return fmt.Errorf("创建数据目录 %s: %w", dir, err)
		}
	}
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return fmt.Errorf("打开迁移连接: %w", err)
	}
	defer db.Close()

	driver, err := sqlite.WithInstance(db, &sqlite.Config{})
	if err != nil {
		return fmt.Errorf("初始化迁移驱动: %w", err)
	}
	src, err := iofs.New(embeddedMigrations, "migrations")
	if err != nil {
		return fmt.Errorf("读取迁移文件: %w", err)
	}
	m, err := migrate.NewWithInstance("iofs", src, "sqlite", driver)
	if err != nil {
		return err
	}
	return fn(m)
}

func MigrateUp(dsn string) error {
	return withMigrator(dsn, func(m *migrate.Migrate) error {
		if err := m.Up(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
			return err
		}
		return nil
	})
}

func MigrateDown(dsn string, steps int) error {
	return withMigrator(dsn, func(m *migrate.Migrate) error {
		if err := m.Steps(-steps); err != nil && !errors.Is(err, migrate.ErrNoChange) {
			return err
		}
		return nil
	})
}

func MigrateVersion(dsn string) (uint, bool, error) {
	var (
		v     uint
		dirty bool
	)
	err := withMigrator(dsn, func(m *migrate.Migrate) error {
		var err error
		v, dirty, err = m.Version()
		return err
	})
	return v, dirty, err
}
