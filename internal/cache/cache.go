package cache

import (
	"sync"
	"time"
)

type Cache interface {
	Get(key string) (any, bool)
	Set(key string, value any, ttl time.Duration)
}

type memoryEntry struct {
	value   any
	expires time.Time
}

type Memory struct {
	mu      sync.RWMutex
	items   map[string]memoryEntry
	maxSize int
}

func NewMemory(maxSize int) *Memory {
	if maxSize <= 0 {
		maxSize = 10000
	}
	return &Memory{items: make(map[string]memoryEntry), maxSize: maxSize}
}

func (m *Memory) Get(key string) (any, bool) {
	m.mu.RLock()
	e, ok := m.items[key]
	m.mu.RUnlock()
	if !ok {
		return nil, false
	}
	if time.Now().After(e.expires) {
		m.mu.Lock()
		delete(m.items, key)
		m.mu.Unlock()
		return nil, false
	}
	return e.value, true
}

func (m *Memory) Set(key string, value any, ttl time.Duration) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, exists := m.items[key]; !exists && len(m.items) >= m.maxSize {
		for k, v := range m.items {
			if time.Now().After(v.expires) {
				delete(m.items, k)
			}
		}
	}
	m.items[key] = memoryEntry{value: value, expires: time.Now().Add(ttl)}
}

func (m *Memory) Len() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return len(m.items)
}
