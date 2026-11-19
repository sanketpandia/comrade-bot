package config

import (
	"context"
	"encoding/json"
	"errors"
	"sync"
	"testing"
	"time"

	"infinite-experiment/politburo/internal/cache"
)

type memCache struct {
	mu   sync.Mutex
	data map[string][]byte
}

func newMemCache() *memCache {
	return &memCache{data: map[string][]byte{}}
}

func (m *memCache) GetJSON(ctx context.Context, key string, dest any) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	raw, ok := m.data[key]
	if !ok {
		return cache.ErrMiss
	}
	return json.Unmarshal(raw, dest)
}

func (m *memCache) SetJSON(ctx context.Context, key string, value any, _ time.Duration) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	raw, err := json.Marshal(value)
	if err != nil {
		return err
	}
	m.data[key] = raw
	return nil
}

func (m *memCache) Delete(ctx context.Context, key string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.data, key)
	return nil
}

func TestCachedStorePopulateAndWriteThrough(t *testing.T) {
	c := newMemCache()
	store := &CachedStore{cache: c}
	key := cache.KeyVAConfig("va-1", TypeDataConnections)

	block := &Block{
		VAID:       "va-1",
		ConfigType: TypeDataConnections,
		Config:     json.RawMessage(`{"revision":2}`),
		UpdatedAt:  time.Unix(200, 0).UTC(),
	}
	store.setCache(context.Background(), "va-1", TypeDataConnections, block)

	var entry vaConfigCacheEntry
	if err := c.GetJSON(context.Background(), key, &entry); err != nil {
		t.Fatalf("GetJSON() = %v", err)
	}
	if entry.Missing || entry.Block == nil || entry.Block.UpdatedAt.Unix() != 200 {
		t.Fatalf("entry = %+v", entry)
	}

	block.Config = json.RawMessage(`{"revision":3}`)
	block.UpdatedAt = time.Unix(300, 0).UTC()
	store.setCache(context.Background(), "va-1", TypeDataConnections, block)
	_ = c.GetJSON(context.Background(), key, &entry)
	if entry.Block == nil || entry.Block.UpdatedAt.Unix() != 300 {
		t.Fatalf("write-through failed: %+v", entry.Block)
	}
}

func TestCachedStoreDoesNotDowngradeCache(t *testing.T) {
	c := newMemCache()
	store := &CachedStore{cache: c}
	key := cache.KeyVAConfig("va-1", TypeDataConnections)
	newer := &Block{UpdatedAt: time.Unix(300, 0).UTC()}
	store.setCacheIfNewer(context.Background(), key, newer)
	older := &Block{UpdatedAt: time.Unix(100, 0).UTC()}
	store.setCacheIfNewer(context.Background(), key, older)
	var entry vaConfigCacheEntry
	_ = c.GetJSON(context.Background(), key, &entry)
	if entry.Block == nil || entry.Block.UpdatedAt.Unix() != 300 {
		t.Fatalf("cache downgraded: %+v", entry.Block)
	}
}

func TestCachedStoreMissSentinel(t *testing.T) {
	c := newMemCache()
	store := &CachedStore{cache: c}
	key := cache.KeyVAConfig("va-1", TypeMapStats)
	store.setCacheEntry(context.Background(), key, vaConfigCacheEntry{Missing: true})
	var entry vaConfigCacheEntry
	if err := c.GetJSON(context.Background(), key, &entry); err != nil {
		t.Fatalf("GetJSON() = %v", err)
	}
	if !entry.Missing {
		t.Fatal("expected missing sentinel")
	}
}

func TestCachedStoreDeleteClearsKey(t *testing.T) {
	c := newMemCache()
	key := cache.KeyVAConfig("va-1", TypeMapStats)
	_ = c.SetJSON(context.Background(), key, vaConfigCacheEntry{Missing: false, Block: &Block{UpdatedAt: time.Now().UTC()}}, time.Minute)
	if err := c.Delete(context.Background(), key); err != nil {
		t.Fatal(err)
	}
	err := c.GetJSON(context.Background(), key, &vaConfigCacheEntry{})
	if !errors.Is(err, cache.ErrMiss) {
		t.Fatalf("expected miss after delete, got %v", err)
	}
}