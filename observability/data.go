package observability

import (
	"context"

	otellog "go.opentelemetry.io/otel/log"
)

type RedisClientLog struct {
	Operation string
	Command   string
	Addr      string
	DB        int
	Duration  float64
	Err       error
	Attrs     []otellog.KeyValue
}

type MySQLClientLog struct {
	Operation string
	Driver    string
	Duration  float64
	Err       error
	Attrs     []otellog.KeyValue
}

func (p *Provider) EmitRedisClientLog(ctx context.Context, entry RedisClientLog) {
	if p == nil {
		p = New()
	}
	if !p.redisClientLogs {
		return
	}

	record := newRecord("redis.client.request", "Redis client request completed", statusFromError(entry.Err), entry.Duration, entry.Err)
	record.AddAttributes(
		otellog.String("db.system.name", "redis"),
		otellog.String("db.operation.name", entry.Operation),
		otellog.String("db.query.summary", entry.Command),
		otellog.String("server.address", entry.Addr),
		otellog.Int("db.redis.database_index", entry.DB),
	)
	record.AddAttributes(entry.Attrs...)
	addTraceAttributes(ctx, &record)
	p.redisClientLogger.Emit(ctx, record)
}

func (p *Provider) EmitMySQLClientLog(ctx context.Context, entry MySQLClientLog) {
	if p == nil {
		p = New()
	}
	if !p.mysqlClientLogs {
		return
	}

	record := newRecord("mysql.client.request", "MySQL client request completed", statusFromError(entry.Err), entry.Duration, entry.Err)
	record.AddAttributes(
		otellog.String("db.system.name", "mysql"),
		otellog.String("db.operation.name", entry.Operation),
		otellog.String("db.client.driver", entry.Driver),
	)
	record.AddAttributes(entry.Attrs...)
	addTraceAttributes(ctx, &record)
	p.mysqlClientLogger.Emit(ctx, record)
}

func statusFromError(err error) int {
	if err != nil {
		return 500
	}
	return 0
}
