package cache

import (
	"context"
	"testing"

	"github.com/stellhub/stellar/config"
)

func TestCacheUsesBigCacheByDefault(t *testing.T) {
	store, err := NewFromConfig(context.Background(), (&config.Config{
		Cache: &config.CacheConfig{},
	}).Normalize().Cache, nil)
	if err != nil {
		t.Fatalf("new cache: %v", err)
	}
	defer store.Close()

	if got := store.AdapterName(); got != AdapterBigCache {
		t.Fatalf("adapter = %q, want %q", got, AdapterBigCache)
	}
	if err := store.SetString(context.Background(), "demo", "hello"); err != nil {
		t.Fatalf("set string: %v", err)
	}
	value, ok, err := store.GetString(context.Background(), "demo")
	if err != nil {
		t.Fatalf("get string: %v", err)
	}
	if !ok || value != "hello" {
		t.Fatalf("value = %q ok = %v, want hello true", value, ok)
	}
	deleted, err := store.Delete(context.Background(), "demo")
	if err != nil {
		t.Fatalf("delete: %v", err)
	}
	if !deleted {
		t.Fatalf("expected key to be deleted")
	}
	_, ok, err = store.GetString(context.Background(), "demo")
	if err != nil {
		t.Fatalf("get deleted key: %v", err)
	}
	if ok {
		t.Fatalf("expected deleted key to be missing")
	}
}

func TestCacheUsesFreeCacheAdapter(t *testing.T) {
	store, err := NewFromConfig(context.Background(), (&config.Config{
		Cache: &config.CacheConfig{
			Adapter:   AdapterFreeCache,
			TTL:       "1m",
			SizeBytes: 1024 * 1024,
		},
	}).Normalize().Cache, nil)
	if err != nil {
		t.Fatalf("new cache: %v", err)
	}
	defer store.Close()

	if got := store.AdapterName(); got != AdapterFreeCache {
		t.Fatalf("adapter = %q, want %q", got, AdapterFreeCache)
	}
	if err := store.Set(context.Background(), "demo", []byte("hello")); err != nil {
		t.Fatalf("set: %v", err)
	}
	value, ok, err := store.Get(context.Background(), "demo")
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if !ok || string(value) != "hello" {
		t.Fatalf("value = %q ok = %v, want hello true", string(value), ok)
	}
}

func TestCacheRejectsInvalidConfig(t *testing.T) {
	if _, err := NewFromConfig(context.Background(), &config.CacheConfig{Adapter: "unknown"}, nil); err == nil {
		t.Fatalf("expected unsupported adapter error")
	}
	if _, err := NewFromConfig(context.Background(), &config.CacheConfig{TTL: "invalid"}, nil); err == nil {
		t.Fatalf("expected invalid ttl error")
	}
	if _, err := NewFromConfig(context.Background(), &config.CacheConfig{}, nil); err != nil {
		t.Fatalf("expected raw config to use default bigcache adapter: %v", err)
	}
}
