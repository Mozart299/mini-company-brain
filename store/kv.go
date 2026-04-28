// Package store implements the distributed knowledge store.
// See docs/03-consistent-hashing.md and docs/04-replication.md.
package store

import (
	"context"
	"fmt"
	"sync"
	"time"

	"company-brain/pkg/types"
)

// Store is the interface every KV backend must satisfy.
type Store interface {
	Set(ctx context.Context, fact types.Fact) error
	Get(ctx context.Context, key string) (types.Fact, error)
	Delete(ctx context.Context, key string) error
	List(ctx context.Context, prefix string) ([]types.Fact, error)
}

// MemoryStore is an in-memory Store used in Milestone 1 before BadgerDB is added.
type MemoryStore struct {
	mu   sync.RWMutex
	data map[string]types.Fact
}

func NewMemoryStore() *MemoryStore {
	return &MemoryStore{data: make(map[string]types.Fact)}
}

func (m *MemoryStore) Set(_ context.Context, fact types.Fact) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	fact.Version = time.Now().UnixNano()
	fact.CreatedAt = time.Now()
	m.data[fact.Key] = fact
	fmt.Printf("[store] set key=%s version=%d\n", fact.Key, fact.Version)
	return nil
}

func (m *MemoryStore) Get(_ context.Context, key string) (types.Fact, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	f, ok := m.data[key]
	if !ok {
		return types.Fact{}, fmt.Errorf("key not found: %s", key)
	}
	return f, nil
}

func (m *MemoryStore) Delete(_ context.Context, key string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.data, key)
	return nil
}

func (m *MemoryStore) List(_ context.Context, prefix string) ([]types.Fact, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	var out []types.Fact
	for k, v := range m.data {
		if len(prefix) == 0 || len(k) >= len(prefix) && k[:len(prefix)] == prefix {
			out = append(out, v)
		}
	}
	return out, nil
}
