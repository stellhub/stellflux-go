package redis

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/redis/go-redis/extra/redisotel/v9"
	goredis "github.com/redis/go-redis/v9"
	"github.com/stellhub/stellar/config"
	"github.com/stellhub/stellar/observability"
)

const DefaultClientName = "redis"

func NewClientFromConfig(cfg *config.RedisConfig, provider *observability.Provider) (*goredis.Client, error) {
	if cfg == nil {
		return nil, fmt.Errorf("stellar: redis config is required")
	}
	options, err := OptionsFromConfig(cfg)
	if err != nil {
		return nil, err
	}

	client := goredis.NewClient(options)
	if provider == nil {
		provider = observability.New()
	}
	if provider.RedisClientTraceEnabled() {
		if err := redisotel.InstrumentTracing(
			client,
			redisotel.WithTracerProvider(provider.TracerProvider()),
			redisotel.WithDBStatement(false),
		); err != nil {
			_ = client.Close()
			return nil, err
		}
	}
	if provider.RedisClientMetricsEnabled() {
		if err := redisotel.InstrumentMetrics(client, redisotel.WithMeterProvider(provider.MeterProvider())); err != nil {
			_ = client.Close()
			return nil, err
		}
	}
	if provider.RedisClientLogsEnabled() {
		client.AddHook(newLogHook(provider, cfg.Addr, cfg.DB))
	}
	return client, nil
}

func OptionsFromConfig(cfg *config.RedisConfig) (*goredis.Options, error) {
	if cfg == nil {
		return nil, fmt.Errorf("stellar: redis config is required")
	}
	options := &goredis.Options{
		Addr:         valueOrDefault(cfg.Addr, "localhost:6379"),
		Username:     cfg.Username,
		Password:     cfg.Password,
		DB:           cfg.DB,
		ClientName:   cfg.ClientName,
		Protocol:     cfg.Protocol,
		MaxRetries:   cfg.MaxRetries,
		PoolSize:     cfg.PoolSize,
		MinIdleConns: cfg.MinIdleConns,
	}

	if cfg.DialTimeout != "" {
		timeout, err := time.ParseDuration(cfg.DialTimeout)
		if err != nil {
			return nil, fmt.Errorf("stellar: invalid redis dial_timeout %q: %w", cfg.DialTimeout, err)
		}
		options.DialTimeout = timeout
	}
	if cfg.ReadTimeout != "" {
		timeout, err := time.ParseDuration(cfg.ReadTimeout)
		if err != nil {
			return nil, fmt.Errorf("stellar: invalid redis read_timeout %q: %w", cfg.ReadTimeout, err)
		}
		options.ReadTimeout = timeout
	}
	if cfg.WriteTimeout != "" {
		timeout, err := time.ParseDuration(cfg.WriteTimeout)
		if err != nil {
			return nil, fmt.Errorf("stellar: invalid redis write_timeout %q: %w", cfg.WriteTimeout, err)
		}
		options.WriteTimeout = timeout
	}
	return options, nil
}

type logHook struct {
	provider *observability.Provider
	addr     string
	db       int
}

func newLogHook(provider *observability.Provider, addr string, db int) goredis.Hook {
	return logHook{
		provider: provider,
		addr:     addr,
		db:       db,
	}
}

func (h logHook) DialHook(next goredis.DialHook) goredis.DialHook {
	return next
}

func (h logHook) ProcessHook(next goredis.ProcessHook) goredis.ProcessHook {
	return func(ctx context.Context, cmd goredis.Cmder) error {
		start := time.Now()
		err := next(ctx, cmd)
		h.emit(ctx, cmd.FullName(), cmd.Name(), time.Since(start), redisLogError(err))
		return err
	}
}

func (h logHook) ProcessPipelineHook(next goredis.ProcessPipelineHook) goredis.ProcessPipelineHook {
	return func(ctx context.Context, cmds []goredis.Cmder) error {
		start := time.Now()
		err := next(ctx, cmds)
		h.emit(ctx, pipelineOperation(cmds), "pipeline", time.Since(start), redisLogError(err))
		return err
	}
}

func (h logHook) emit(ctx context.Context, operation string, command string, duration time.Duration, err error) {
	h.provider.EmitRedisClientLog(ctx, observability.RedisClientLog{
		Operation: operation,
		Command:   command,
		Addr:      h.addr,
		DB:        h.db,
		Duration:  duration.Seconds(),
		Err:       err,
	})
}

func pipelineOperation(cmds []goredis.Cmder) string {
	if len(cmds) == 0 {
		return "pipeline"
	}
	names := make([]string, 0, len(cmds))
	for _, cmd := range cmds {
		if cmd == nil || strings.TrimSpace(cmd.FullName()) == "" {
			continue
		}
		names = append(names, cmd.FullName())
	}
	if len(names) == 0 {
		return "pipeline"
	}
	return strings.Join(names, ",")
}

func redisLogError(err error) error {
	if errors.Is(err, goredis.Nil) {
		return nil
	}
	return err
}

func valueOrDefault(value string, fallback string) string {
	if strings.TrimSpace(value) != "" {
		return value
	}
	return fallback
}
