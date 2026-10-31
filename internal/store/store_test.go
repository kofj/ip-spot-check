package store

import (
	"context"
	"testing"
	"time"
)

func openTestStore(t *testing.T) (*Store, string) {
	t.Helper()
	dsn := "file:" + t.TempDir() + "/test.db?_pragma=journal_mode(WAL)"
	if err := MigrateUp(dsn); err != nil {
		t.Fatalf("MigrateUp: %v", err)
	}
	st, err := Open("sqlite", dsn, 1)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	t.Cleanup(func() { st.Close() })
	return st, dsn
}

func TestMigrateUpIsIdempotent(t *testing.T) {
	_, dsn := openTestStore(t)
	if err := MigrateUp(dsn); err != nil {
		t.Fatalf("重复 MigrateUp 应无错误: %v", err)
	}
	v, dirty, err := MigrateVersion(dsn)
	if err != nil {
		t.Fatalf("Version: %v", err)
	}
	if dirty {
		t.Error("不应处于 dirty 状态")
	}
	if v != 1 {
		t.Errorf("version = %d, want 1", v)
	}
}

func TestMigrateDownRemovesTables(t *testing.T) {
	st, dsn := openTestStore(t)
	if err := MigrateDown(dsn, 1); err != nil {
		t.Fatalf("MigrateDown: %v", err)
	}
	if _, err := st.DB().Exec(`SELECT * FROM observations`); err == nil {
		t.Fatal("回滚后 observations 表应不存在")
	}
}

func TestTablesCreated(t *testing.T) {
	st, _ := openTestStore(t)
	for _, table := range []string{"observations", "queries", "quota_usage"} {
		var name string
		err := st.DB().QueryRow(
			`SELECT name FROM sqlite_master WHERE type='table' AND name=?`, table).Scan(&name)
		if err != nil {
			t.Errorf("表 %s 未创建: %v", table, err)
		}
	}
}

func TestSaveAndCountQuery(t *testing.T) {
	st, _ := openTestStore(t)
	ctx := context.Background()

	if err := st.SaveQuery(ctx, "1.1.1.1", time.Now()); err != nil {
		t.Fatalf("SaveQuery: %v", err)
	}
	if err := st.SaveQuery(ctx, "1.1.1.1", time.Now()); err != nil {
		t.Fatalf("SaveQuery: %v", err)
	}
	n, err := st.QueryCount(ctx, "1.1.1.1")
	if err != nil {
		t.Fatalf("QueryCount: %v", err)
	}
	if n != 2 {
		t.Errorf("count = %d, want 2", n)
	}
}
