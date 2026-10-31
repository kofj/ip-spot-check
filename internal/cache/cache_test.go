package cache

import (
	"testing"
	"time"
)

func TestSetGet(t *testing.T) {
	c := NewMemory(10)
	c.Set("a", 1, time.Minute)
	got, ok := c.Get("a")
	if !ok || got != 1 {
		t.Fatalf("got %v ok=%v", got, ok)
	}
}

func TestExpiredEntryMisses(t *testing.T) {
	c := NewMemory(10)
	c.Set("a", 1, -time.Second)
	if _, ok := c.Get("a"); ok {
		t.Fatal("过期条目应未命中")
	}
}

func TestEvictsExpiredWhenFull(t *testing.T) {
	c := NewMemory(2)
	c.Set("expired", 1, -time.Second)
	c.Set("live", 2, time.Minute)
	c.Set("new", 3, time.Minute)

	if _, ok := c.Get("new"); !ok {
		t.Error("新条目应存在")
	}
	if c.Len() > 2 {
		t.Errorf("长度 %d 超出上限", c.Len())
	}
}
