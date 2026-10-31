package quota

import (
	"context"
	"testing"
	"time"

	"github.com/kofj/ip-spot-check/internal/store"
)

func newGuard(t *testing.T, limit int) (*SQLGuard, *store.Store) {
	t.Helper()
	dsn := "file:" + t.TempDir() + "/q.db"
	if err := store.MigrateUp(dsn); err != nil {
		t.Fatalf("MigrateUp: %v", err)
	}
	st, err := store.Open("sqlite", dsn, 1)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	t.Cleanup(func() { st.Close() })
	g := NewSQLGuard(st.DB(), func(string) int { return limit })
	return g, st
}

func TestAcquireWithinLimit(t *testing.T) {
	g, _ := newGuard(t, 2)
	ctx := context.Background()

	if err := g.Acquire(ctx, "abuseipdb"); err != nil {
		t.Fatalf("第 1 次: %v", err)
	}
	if err := g.Acquire(ctx, "abuseipdb"); err != nil {
		t.Fatalf("第 2 次: %v", err)
	}
	used, limit := g.Usage(ctx, "abuseipdb")
	if used != 2 || limit != 2 {
		t.Errorf("used=%d limit=%d", used, limit)
	}
}

func TestAcquireExceedsLimit(t *testing.T) {
	g, _ := newGuard(t, 1)
	ctx := context.Background()

	if err := g.Acquire(ctx, "abuseipdb"); err != nil {
		t.Fatalf("第 1 次: %v", err)
	}
	if err := g.Acquire(ctx, "abuseipdb"); err != ErrExceeded {
		t.Fatalf("期望 ErrExceeded，得到 %v", err)
	}
}

func TestQuotaPersistsAcrossRestart(t *testing.T) {
	dir := t.TempDir()
	dsn := "file:" + dir + "/q.db"

	if err := store.MigrateUp(dsn); err != nil {
		t.Fatal(err)
	}
	st, err := store.Open("sqlite", dsn, 1)
	if err != nil {
		t.Fatal(err)
	}
	g := NewSQLGuard(st.DB(), func(string) int { return 5 })
	if err := g.Acquire(context.Background(), "rdap"); err != nil {
		t.Fatal(err)
	}
	st.Close()

	st2, err := store.Open("sqlite", dsn, 1)
	if err != nil {
		t.Fatal(err)
	}
	defer st2.Close()
	g2 := NewSQLGuard(st2.DB(), func(string) int { return 5 })
	if used, _ := g2.Usage(context.Background(), "rdap"); used != 1 {
		t.Errorf("重启后 used = %d, want 1", used)
	}
}

func TestQuotaResetsNextDay(t *testing.T) {
	g, _ := newGuard(t, 1)
	ctx := context.Background()

	day1 := time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)
	g.now = func() time.Time { return day1 }
	if err := g.Acquire(ctx, "rdap"); err != nil {
		t.Fatal(err)
	}

	g.now = func() time.Time { return day1.Add(24 * time.Hour) }
	if err := g.Acquire(ctx, "rdap"); err != nil {
		t.Fatalf("次日应重置: %v", err)
	}
}
