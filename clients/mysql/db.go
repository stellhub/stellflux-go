package mysql

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/XSAM/otelsql"
	_ "github.com/go-sql-driver/mysql"
	"github.com/stellhub/stellar/config"
	"github.com/stellhub/stellar/observability"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
	metricnoop "go.opentelemetry.io/otel/metric/noop"
	tracenoop "go.opentelemetry.io/otel/trace/noop"
)

const DefaultDBName = "mysql"

type DB struct {
	*sql.DB
	driver            string
	provider          *observability.Provider
	logsEnabled       bool
	statsRegistration metric.Registration
}

func NewDBFromConfig(ctx context.Context, cfg *config.MySQLConfig, provider *observability.Provider) (*DB, error) {
	if cfg == nil {
		return nil, fmt.Errorf("stellar: mysql config is required")
	}
	if strings.TrimSpace(cfg.DSN) == "" {
		return nil, fmt.Errorf("stellar: mysql dsn is required")
	}
	if provider == nil {
		provider = observability.New()
	}

	driver := valueOrDefault(cfg.Driver, "mysql")
	sqlDB, err := openDB(driver, cfg.DSN, provider)
	if err != nil {
		return nil, err
	}
	if err := applyPoolConfig(sqlDB, cfg); err != nil {
		_ = sqlDB.Close()
		return nil, err
	}

	var registration metric.Registration
	if provider.MySQLClientMetricsEnabled() {
		registration, err = otelsql.RegisterDBStatsMetrics(
			sqlDB,
			otelsql.WithMeterProvider(provider.MeterProvider()),
			otelsql.WithAttributes(attribute.String("db.system.name", "mysql")),
		)
		if err != nil {
			_ = sqlDB.Close()
			return nil, err
		}
	}

	db := &DB{
		DB:                sqlDB,
		driver:            driver,
		provider:          provider,
		logsEnabled:       provider.MySQLClientLogsEnabled(),
		statsRegistration: registration,
	}
	if cfg.PingOnStartup {
		pingCtx := ctx
		if pingCtx == nil {
			pingCtx = context.Background()
		}
		if cfg.PingTimeout != "" {
			timeout, err := time.ParseDuration(cfg.PingTimeout)
			if err != nil {
				_ = db.Close()
				return nil, fmt.Errorf("stellar: invalid mysql ping_timeout %q: %w", cfg.PingTimeout, err)
			}
			var cancel context.CancelFunc
			pingCtx, cancel = context.WithTimeout(pingCtx, timeout)
			defer cancel()
		}
		if err := db.PingContext(pingCtx); err != nil {
			_ = db.Close()
			return nil, err
		}
	}
	return db, nil
}

func (db *DB) Close() error {
	if db == nil || db.DB == nil {
		return nil
	}
	if db.statsRegistration != nil {
		_ = db.statsRegistration.Unregister()
	}
	return db.DB.Close()
}

func (db *DB) SQLDB() *sql.DB {
	if db == nil {
		return nil
	}
	return db.DB
}

func (db *DB) PingContext(ctx context.Context) error {
	start := time.Now()
	err := db.DB.PingContext(ctx)
	db.emit(ctx, "ping", time.Since(start), err)
	return err
}

func (db *DB) ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error) {
	start := time.Now()
	result, err := db.DB.ExecContext(ctx, query, args...)
	db.emit(ctx, operationFromQuery(query, "exec"), time.Since(start), err)
	return result, err
}

func (db *DB) QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error) {
	start := time.Now()
	rows, err := db.DB.QueryContext(ctx, query, args...)
	db.emit(ctx, operationFromQuery(query, "query"), time.Since(start), err)
	return rows, err
}

func (db *DB) QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row {
	start := time.Now()
	row := db.DB.QueryRowContext(ctx, query, args...)
	db.emit(ctx, operationFromQuery(query, "query_row"), time.Since(start), nil)
	return row
}

func (db *DB) PrepareContext(ctx context.Context, query string) (*sql.Stmt, error) {
	start := time.Now()
	stmt, err := db.DB.PrepareContext(ctx, query)
	db.emit(ctx, "prepare", time.Since(start), err)
	return stmt, err
}

func (db *DB) BeginTx(ctx context.Context, opts *sql.TxOptions) (*sql.Tx, error) {
	start := time.Now()
	tx, err := db.DB.BeginTx(ctx, opts)
	db.emit(ctx, "begin", time.Since(start), err)
	return tx, err
}

func (db *DB) emit(ctx context.Context, operation string, duration time.Duration, err error) {
	if db == nil || !db.logsEnabled {
		return
	}
	db.provider.EmitMySQLClientLog(ctx, observability.MySQLClientLog{
		Operation: operation,
		Driver:    db.driver,
		Duration:  duration.Seconds(),
		Err:       err,
	})
}

func openDB(driver string, dsn string, provider *observability.Provider) (*sql.DB, error) {
	if !provider.MySQLClientTraceEnabled() && !provider.MySQLClientMetricsEnabled() {
		return sql.Open(driver, dsn)
	}

	options := []otelsql.Option{
		otelsql.WithTextMapPropagator(provider.Propagator()),
		otelsql.WithAttributes(attribute.String("db.system.name", "mysql")),
	}
	if provider.MySQLClientTraceEnabled() {
		options = append(options, otelsql.WithTracerProvider(provider.TracerProvider()))
	} else {
		options = append(options, otelsql.WithTracerProvider(tracenoop.NewTracerProvider()))
	}
	if provider.MySQLClientMetricsEnabled() {
		options = append(options, otelsql.WithMeterProvider(provider.MeterProvider()))
	} else {
		options = append(options, otelsql.WithMeterProvider(metricnoop.NewMeterProvider()))
	}
	return otelsql.Open(driver, dsn, options...)
}

func applyPoolConfig(db *sql.DB, cfg *config.MySQLConfig) error {
	if cfg.MaxOpenConns > 0 {
		db.SetMaxOpenConns(cfg.MaxOpenConns)
	}
	if cfg.MaxIdleConns > 0 {
		db.SetMaxIdleConns(cfg.MaxIdleConns)
	}
	if cfg.ConnMaxLifetime != "" {
		duration, err := time.ParseDuration(cfg.ConnMaxLifetime)
		if err != nil {
			return fmt.Errorf("stellar: invalid mysql conn_max_lifetime %q: %w", cfg.ConnMaxLifetime, err)
		}
		db.SetConnMaxLifetime(duration)
	}
	if cfg.ConnMaxIdleTime != "" {
		duration, err := time.ParseDuration(cfg.ConnMaxIdleTime)
		if err != nil {
			return fmt.Errorf("stellar: invalid mysql conn_max_idle_time %q: %w", cfg.ConnMaxIdleTime, err)
		}
		db.SetConnMaxIdleTime(duration)
	}
	return nil
}

func operationFromQuery(query string, fallback string) string {
	query = strings.TrimSpace(query)
	if query == "" {
		return fallback
	}
	fields := strings.Fields(query)
	if len(fields) == 0 {
		return fallback
	}
	return strings.ToLower(fields[0])
}

func valueOrDefault(value string, fallback string) string {
	if strings.TrimSpace(value) != "" {
		return value
	}
	return fallback
}
