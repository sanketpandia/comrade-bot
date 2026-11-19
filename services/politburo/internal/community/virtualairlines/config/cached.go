package config

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"time"

	"infinite-experiment/politburo/internal/cache"
)

// vaConfigCacheEntry is the single Redis value per VA config type.
// Either Missing is true or Block holds the row snapshot.
type vaConfigCacheEntry struct {
	Missing bool  `json:"missing"`
	Block   *Block `json:"block,omitempty"`
}

// CachedStore wraps a Repository with Redis read-through caching.
type CachedStore struct {
	repo  *Repository
	cache cacheStore
}

type cacheStore interface {
	GetJSON(context.Context, string, any) error
	SetJSON(context.Context, string, any, time.Duration) error
	Delete(context.Context, string) error
}

func NewCachedStore(repo *Repository, c cacheStore) *CachedStore {
	return &CachedStore{repo: repo, cache: c}
}

func (s *CachedStore) Get(ctx context.Context, vaID, configType string) (*Block, error) {
	if s == nil || s.repo == nil {
		return nil, errors.New("config: cached store not configured")
	}
	key := cache.KeyVAConfig(vaID, configType)
	if s.cache != nil {
		var entry vaConfigCacheEntry
		err := s.cache.GetJSON(ctx, key, &entry)
		if err == nil {
			if entry.Missing {
				return nil, ErrNotFound
			}
			if entry.Block != nil {
				return entry.Block, nil
			}
		} else if !errors.Is(err, cache.ErrMiss) {
			slog.WarnContext(ctx, "va config cache get failed", "key", key, "error", err)
		}
	}
	block, err := s.repo.Get(ctx, vaID, configType)
	if err != nil {
		if errors.Is(err, ErrNotFound) && s.cache != nil {
			s.setCacheEntry(ctx, key, vaConfigCacheEntry{Missing: true})
		}
		return nil, err
	}
	s.setCacheIfNewer(ctx, key, block)
	return block, nil
}

func (s *CachedStore) UpsertIfRevision(ctx context.Context, vaID, configType string, config json.RawMessage, expectedRevision int, updatedBy string) (*Block, error) {
	block, err := s.repo.UpsertIfRevision(ctx, vaID, configType, config, expectedRevision, updatedBy)
	if err != nil {
		return nil, err
	}
	s.setCache(ctx, vaID, configType, block)
	return block, nil
}

func (s *CachedStore) Upsert(ctx context.Context, vaID, configType string, config json.RawMessage, updatedBy string) (*Block, error) {
	block, err := s.repo.Upsert(ctx, vaID, configType, config, updatedBy)
	if err != nil {
		return nil, err
	}
	s.setCache(ctx, vaID, configType, block)
	return block, nil
}

func (s *CachedStore) Delete(ctx context.Context, vaID, configType string) error {
	if err := s.repo.Delete(ctx, vaID, configType); err != nil {
		return err
	}
	if s.cache != nil {
		key := cache.KeyVAConfig(vaID, configType)
		if err := s.cache.Delete(ctx, key); err != nil {
			slog.WarnContext(ctx, "va config cache delete failed", "key", key, "error", err)
		}
	}
	return nil
}

func (s *CachedStore) setCache(ctx context.Context, vaID, configType string, block *Block) {
	if s.cache == nil || block == nil {
		return
	}
	key := cache.KeyVAConfig(vaID, configType)
	s.setCacheEntry(ctx, key, vaConfigCacheEntry{Missing: false, Block: block})
}

func (s *CachedStore) setCacheIfNewer(ctx context.Context, key string, block *Block) {
	if s.cache == nil || block == nil {
		return
	}
	var entry vaConfigCacheEntry
	err := s.cache.GetJSON(ctx, key, &entry)
	if err == nil && !entry.Missing && entry.Block != nil {
		if !block.UpdatedAt.After(entry.Block.UpdatedAt) {
			return
		}
	} else if err != nil && !errors.Is(err, cache.ErrMiss) {
		slog.WarnContext(ctx, "va config cache read before populate failed", "key", key, "error", err)
	}
	s.setCacheEntry(ctx, key, vaConfigCacheEntry{Missing: false, Block: block})
}

func (s *CachedStore) setCacheEntry(ctx context.Context, key string, entry vaConfigCacheEntry) {
	if err := s.cache.SetJSON(ctx, key, entry, cache.ConfigCacheTTL); err != nil {
		slog.WarnContext(ctx, "va config cache set failed", "key", key, "error", err)
	}
}