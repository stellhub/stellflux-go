package cache

import (
	"context"
	"errors"
	"time"

	bigcachelib "github.com/allegro/bigcache/v3"
	"github.com/stellhub/stellar/config"
)

type bigCacheAdapter struct {
	cache *bigcachelib.BigCache
}

func newBigCacheAdapter(ctx context.Context, cfg *config.CacheConfig) (Adapter, error) {
	ttl, err := ttlFromConfig(cfg)
	if err != nil {
		return nil, err
	}
	cleanWindow, err := durationFromConfig(cfg.CleanWindow, "clean_window", time.Minute)
	if err != nil {
		return nil, err
	}

	bigCacheConfig := bigcachelib.DefaultConfig(ttl)
	bigCacheConfig.CleanWindow = cleanWindow
	if cfg.Shards > 0 {
		bigCacheConfig.Shards = cfg.Shards
	}
	if cfg.MaxEntriesInWindow > 0 {
		bigCacheConfig.MaxEntriesInWindow = cfg.MaxEntriesInWindow
	}
	if cfg.MaxEntrySize > 0 {
		bigCacheConfig.MaxEntrySize = cfg.MaxEntrySize
	}
	if cfg.HardMaxCacheSizeMB > 0 {
		bigCacheConfig.HardMaxCacheSize = cfg.HardMaxCacheSizeMB
	}
	if cfg.StatsEnabled == nil {
		bigCacheConfig.StatsEnabled = true
	} else {
		bigCacheConfig.StatsEnabled = *cfg.StatsEnabled
	}
	bigCacheConfig.Verbose = cfg.Verbose

	instance, err := bigcachelib.New(ctx, bigCacheConfig)
	if err != nil {
		return nil, err
	}
	return &bigCacheAdapter{cache: instance}, nil
}

func (a *bigCacheAdapter) Name() string {
	return AdapterBigCache
}

func (a *bigCacheAdapter) Get(_ context.Context, key string) ([]byte, error) {
	value, err := a.cache.Get(key)
	if err != nil {
		return nil, bigCacheError(err)
	}
	return value, nil
}

func (a *bigCacheAdapter) Set(_ context.Context, key string, value []byte) error {
	return a.cache.Set(key, value)
}

func (a *bigCacheAdapter) Delete(_ context.Context, key string) (bool, error) {
	err := a.cache.Delete(key)
	if errors.Is(err, bigcachelib.ErrEntryNotFound) {
		return false, nil
	}
	return err == nil, err
}

func (a *bigCacheAdapter) Len() int64 {
	if a == nil || a.cache == nil {
		return 0
	}
	return int64(a.cache.Len())
}

func (a *bigCacheAdapter) Capacity() int64 {
	if a == nil || a.cache == nil {
		return 0
	}
	return int64(a.cache.Capacity())
}

func (a *bigCacheAdapter) Close() error {
	if a == nil || a.cache == nil {
		return nil
	}
	return a.cache.Close()
}

func bigCacheError(err error) error {
	if errors.Is(err, bigcachelib.ErrEntryNotFound) {
		return ErrNotFound
	}
	return err
}
