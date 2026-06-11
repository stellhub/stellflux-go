package redis

import (
	"testing"

	"github.com/stellhub/stellar/config"
)

func TestOptionsFromConfig(t *testing.T) {
	options, err := OptionsFromConfig(&config.RedisConfig{
		Addr:         "localhost:6379",
		Username:     "default",
		Password:     "secret",
		DB:           2,
		ClientName:   "stellar-test",
		PoolSize:     16,
		MinIdleConns: 2,
		DialTimeout:  "2s",
		ReadTimeout:  "1s",
		WriteTimeout: "1s",
	})
	if err != nil {
		t.Fatalf("options from config: %v", err)
	}
	if options.Addr != "localhost:6379" {
		t.Fatalf("unexpected addr %q", options.Addr)
	}
	if options.DB != 2 {
		t.Fatalf("unexpected db %d", options.DB)
	}
	if options.PoolSize != 16 {
		t.Fatalf("unexpected pool size %d", options.PoolSize)
	}
}

func TestOptionsFromConfigRejectsInvalidDuration(t *testing.T) {
	_, err := OptionsFromConfig(&config.RedisConfig{
		DialTimeout: "soon",
	})
	if err == nil {
		t.Fatalf("expected invalid duration error")
	}
}
