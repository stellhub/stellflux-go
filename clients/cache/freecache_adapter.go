package cache

import (
	"context"
	"errors"
	"math"

	freecachelib "github.com/coocood/freecache"
	"github.com/stellhub/stellar/config"
)

type freeCacheAdapter struct {
	cache         *freecachelib.Cache
	sizeBytes     int
	expireSeconds int
}

func newFreeCacheAdapter(cfg *config.CacheConfig) (Adapter, error) {
	ttl, err := ttlFromConfig(cfg)
	if err != nil {
		return nil, err
	}
	sizeBytes := cfg.SizeBytes
	if sizeBytes <= 0 {
		sizeBytes = 64 * 1024 * 1024
	}
	expireSeconds := 0
	if ttl > 0 {
		if ttl.Seconds() > float64(math.MaxInt) {
			expireSeconds = math.MaxInt
		} else {
			expireSeconds = int(ttl.Seconds())
			if expireSeconds == 0 {
				expireSeconds = 1
			}
		}
	}
	return &freeCacheAdapter{
		cache:         freecachelib.NewCache(sizeBytes),
		sizeBytes:     sizeBytes,
		expireSeconds: expireSeconds,
	}, nil
}

func (a *freeCacheAdapter) Name() string {
	return AdapterFreeCache
}

func (a *freeCacheAdapter) Get(_ context.Context, key string) ([]byte, error) {
	value, err := a.cache.Get([]byte(key))
	if errors.Is(err, freecachelib.ErrNotFound) {
		return nil, ErrNotFound
	}
	return value, err
}

func (a *freeCacheAdapter) Set(_ context.Context, key string, value []byte) error {
	return a.cache.Set([]byte(key), value, a.expireSeconds)
}

func (a *freeCacheAdapter) Delete(_ context.Context, key string) (bool, error) {
	return a.cache.Del([]byte(key)), nil
}

func (a *freeCacheAdapter) Len() int64 {
	if a == nil || a.cache == nil {
		return 0
	}
	return a.cache.EntryCount()
}

func (a *freeCacheAdapter) Capacity() int64 {
	if a == nil {
		return 0
	}
	return int64(a.sizeBytes)
}

func (a *freeCacheAdapter) Close() error {
	if a == nil || a.cache == nil {
		return nil
	}
	a.cache.Clear()
	return nil
}
