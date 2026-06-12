package cache

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/stellhub/stellar/config"
	"github.com/stellhub/stellar/observability"
)

const (
	DefaultName      = "cache"
	AdapterBigCache  = "bigcache"
	AdapterFreeCache = "freecache"
)

var ErrNotFound = errors.New("stellar: cache key not found")

type Adapter interface {
	Name() string
	Get(context.Context, string) ([]byte, error)
	Set(context.Context, string, []byte) error
	Delete(context.Context, string) (bool, error)
	Len() int64
	Capacity() int64
	Close() error
}

type Cache struct {
	adapter  Adapter
	provider *observability.Provider
}

func New(adapter Adapter, provider *observability.Provider) (*Cache, error) {
	if adapter == nil {
		return nil, fmt.Errorf("stellar: cache adapter is required")
	}
	if provider == nil {
		provider = observability.New()
	}
	return &Cache{
		adapter:  adapter,
		provider: provider,
	}, nil
}

func NewFromConfig(ctx context.Context, cfg *config.CacheConfig, provider *observability.Provider) (*Cache, error) {
	if cfg == nil {
		return nil, fmt.Errorf("stellar: cache config is required")
	}
	if provider == nil {
		provider = observability.New()
	}
	if ctx == nil {
		ctx = context.Background()
	}

	adapter, err := newAdapter(ctx, cfg)
	if err != nil {
		return nil, err
	}
	return New(adapter, provider)
}

func (c *Cache) AdapterName() string {
	if c == nil || c.adapter == nil {
		return ""
	}
	return c.adapter.Name()
}

func (c *Cache) Get(ctx context.Context, key string) ([]byte, bool, error) {
	if err := validateKey(key); err != nil {
		return nil, false, err
	}
	if err := c.validate(); err != nil {
		return nil, false, err
	}
	ctx = contextOrBackground(ctx)
	ctx, finish := c.provider.StartCacheClient(ctx, observability.CacheClientRequest{
		Adapter:   c.adapter.Name(),
		Operation: "get",
	})

	value, err := c.adapter.Get(ctx, key)
	result := observability.CacheClientResult{
		Result:        observability.CacheResultHit,
		SizeBytes:     len(value),
		Entries:       c.adapter.Len(),
		CapacityBytes: c.adapter.Capacity(),
		Err:           err,
	}
	if errors.Is(err, ErrNotFound) {
		result.Result = observability.CacheResultMiss
		result.Err = nil
		result.SizeBytes = -1
		finish(result)
		return nil, false, nil
	}
	if err != nil {
		result.Result = observability.CacheResultError
		finish(result)
		return nil, false, err
	}
	finish(result)
	return value, true, nil
}

func (c *Cache) GetString(ctx context.Context, key string) (string, bool, error) {
	value, ok, err := c.Get(ctx, key)
	if err != nil || !ok {
		return "", ok, err
	}
	return string(value), true, nil
}

func (c *Cache) Set(ctx context.Context, key string, value []byte) error {
	if err := validateKey(key); err != nil {
		return err
	}
	if err := c.validate(); err != nil {
		return err
	}
	ctx = contextOrBackground(ctx)
	ctx, finish := c.provider.StartCacheClient(ctx, observability.CacheClientRequest{
		Adapter:   c.adapter.Name(),
		Operation: "set",
	})

	err := c.adapter.Set(ctx, key, value)
	result := observability.CacheClientResult{
		Result:        observability.CacheResultOK,
		SizeBytes:     len(value),
		Entries:       c.adapter.Len(),
		CapacityBytes: c.adapter.Capacity(),
		Err:           err,
	}
	if err != nil {
		result.Result = observability.CacheResultError
	}
	finish(result)
	return err
}

func (c *Cache) SetString(ctx context.Context, key string, value string) error {
	return c.Set(ctx, key, []byte(value))
}

func (c *Cache) Delete(ctx context.Context, key string) (bool, error) {
	if err := validateKey(key); err != nil {
		return false, err
	}
	if err := c.validate(); err != nil {
		return false, err
	}
	ctx = contextOrBackground(ctx)
	ctx, finish := c.provider.StartCacheClient(ctx, observability.CacheClientRequest{
		Adapter:   c.adapter.Name(),
		Operation: "delete",
	})

	deleted, err := c.adapter.Delete(ctx, key)
	result := observability.CacheClientResult{
		Result:        observability.CacheResultOK,
		SizeBytes:     -1,
		Entries:       c.adapter.Len(),
		CapacityBytes: c.adapter.Capacity(),
		Err:           err,
	}
	if err != nil {
		result.Result = observability.CacheResultError
	} else if !deleted {
		result.Result = observability.CacheResultMiss
	}
	finish(result)
	return deleted, err
}

func (c *Cache) Len() int64 {
	if c == nil || c.adapter == nil {
		return 0
	}
	return c.adapter.Len()
}

func (c *Cache) Capacity() int64 {
	if c == nil || c.adapter == nil {
		return 0
	}
	return c.adapter.Capacity()
}

func (c *Cache) Close() error {
	if c == nil || c.adapter == nil {
		return nil
	}
	return c.adapter.Close()
}

func newAdapter(ctx context.Context, cfg *config.CacheConfig) (Adapter, error) {
	switch normalizeAdapter(cfg.Adapter) {
	case AdapterFreeCache:
		return newFreeCacheAdapter(cfg)
	case AdapterBigCache:
		return newBigCacheAdapter(ctx, cfg)
	default:
		return nil, fmt.Errorf("stellar: unsupported cache adapter %q", cfg.Adapter)
	}
}

func normalizeAdapter(adapter string) string {
	if strings.TrimSpace(adapter) == "" {
		return AdapterBigCache
	}
	return strings.ToLower(strings.TrimSpace(adapter))
}

func ttlFromConfig(cfg *config.CacheConfig) (time.Duration, error) {
	ttl := strings.TrimSpace(cfg.TTL)
	if ttl == "" {
		return 10 * time.Minute, nil
	}
	duration, err := time.ParseDuration(ttl)
	if err != nil {
		return 0, fmt.Errorf("stellar: invalid cache ttl %q: %w", cfg.TTL, err)
	}
	return duration, nil
}

func durationFromConfig(value string, name string, fallback time.Duration) (time.Duration, error) {
	if strings.TrimSpace(value) == "" {
		return fallback, nil
	}
	duration, err := time.ParseDuration(value)
	if err != nil {
		return 0, fmt.Errorf("stellar: invalid cache %s %q: %w", name, value, err)
	}
	return duration, nil
}

func contextOrBackground(ctx context.Context) context.Context {
	if ctx == nil {
		return context.Background()
	}
	return ctx
}

func validateKey(key string) error {
	if strings.TrimSpace(key) == "" {
		return fmt.Errorf("stellar: cache key is required")
	}
	return nil
}

func (c *Cache) validate() error {
	if c == nil || c.adapter == nil {
		return fmt.Errorf("stellar: cache is not configured")
	}
	if c.provider == nil {
		c.provider = observability.New()
	}
	return nil
}
