<p align="right">
English | <a href="README.zh-CN.md">简体中文</a>
</p>

# Stellar

`stellar` is the Go framework for the Stell middleware ecosystem. It provides a unified application foundation for services that need to integrate Stell standards for configuration, discovery, messaging, observability, governance, and platform operations.

It has the same positioning as [`stellhub/stellflux`](https://github.com/stellhub/stellflux), while following Go conventions: small packages, explicit composition, context propagation, standard library first, and predictable lifecycle management.

## Positioning

Stellar is not a middleware server and does not implement business logic. It is a framework layer for Go services that need a consistent way to adopt Stell middleware capabilities.

## Core Responsibilities

- Provide unified application configuration and runtime metadata.
- Define a lightweight module lifecycle for Stell middleware integrations.
- Standardize how Go services connect to StellMap, StellFlow, StellNula, StellSpec, StellOrbit, StellGate, and StellAtlas.
- Expose framework status for health checks, control planes, and platform consoles.
- Keep observability, service identity, environment, and zone metadata consistent across services.

## Middleware Standards

| Standard | Responsibility |
| --- | --- |
| StellMap | Service discovery and registry integration |
| StellFlow | Messaging and event streaming integration |
| StellNula | Configuration center integration |
| StellSpec | Observability and log query standard integration |
| StellOrbit | Traffic governance, routing, retries, and lifecycle policy integration |
| StellGate | API gateway and ingress standard integration |
| StellAtlas | CMDB, asset inventory, topology, and lifecycle metadata integration |

## Current Status

| Item | Value |
| --- | --- |
| Stability | Early development |
| Language | Go |
| Project type | Go framework |
| Target users | Go microservices, platform services, infrastructure components |
| Maintainer | StellHub |

## Quick Start

Install the module:

```bash
go get github.com/stellhub/stellar
```

Create an application:

```go
package main

import (
	"log"

	"github.com/stellhub/stellar"
)

func main() {
	if err := stellar.Run(); err != nil {
		log.Fatal(err)
	}
}
```

Add `application.yml`:

```yaml
app:
  name: example-service
  env: dev
  zone: local
http:
  server:
    enabled: true
    port: 8080
    adapter: gin
    observability:
      trace: true
      metrics: true
      logs: true
  client:
    enabled: true
    timeout: 3s
    max_idle_conns: 100
    max_idle_conns_per_host: 10
    idle_conn_timeout: 90s
    observability:
      trace: true
      metrics: true
      logs: false
    clients:
      user-service:
        base_url: http://localhost:8081
        timeout: 2s
      order-service:
        base_url: http://localhost:8082
        timeout: 5s
grpc:
  server:
    enabled: true
    port: 9090
    adapter: grpc-go
    observability:
      trace: true
      metrics: true
      logs: true
  client:
    enabled: true
    timeout: 3s
    insecure: true
    observability:
      trace: true
      metrics: true
      logs: false
    clients:
      user-service:
        target: dns:///localhost:9091
        timeout: 2s
      order-service:
        target: dns:///localhost:9092
        timeout: 5s
redis:
  enabled: true
  addr: localhost:6379
  db: 0
  pool_size: 16
  debug_api:
    enabled: true
    prefix: /redis
  observability:
    trace: true
    metrics: true
    logs: true
mysql:
  enabled: true
  dsn: user:password@tcp(localhost:3306)/example?parseTime=true
  max_open_conns: 25
  max_idle_conns: 5
  conn_max_lifetime: 30m
  conn_max_idle_time: 5m
  ping_on_startup: false
  debug_api:
    enabled: true
    prefix: /mysql
  observability:
    trace: true
    metrics: true
    logs: true
postgresql:
  enabled: true
  dsn: postgres://user:password@localhost:5432/example?sslmode=disable
  max_open_conns: 25
  max_idle_conns: 5
  conn_max_lifetime: 30m
  conn_max_idle_time: 5m
  ping_on_startup: false
  debug_api:
    enabled: true
    prefix: /postgresql
  observability:
    trace: true
    metrics: true
    logs: true
cache:
  enabled: true
  adapter: bigcache
  ttl: 10m
  clean_window: 1m
  shards: 1024
  hard_max_cache_size_mb: 64
  size_bytes: 67108864
  debug_api:
    enabled: true
    prefix: /cache
  observability:
    trace: true
    metrics: true
    logs: true
registry:
  enabled: true
  adapter: stellmap
  endpoints:
    - http://localhost:18090
  namespace: default
  service: example-service
  instance_id: example-service-1
  zone: local
  ttl: 30s
  heartbeat_interval: 10s
  service_endpoints:
    - name: http
      protocol: http
      host: 127.0.0.1
      port: 8080
opentelemetry:
  trace: true
  metrics: true
```

Run the included HTTP server example:

```bash
go run ./examples/http/server/simple
```

Then open:

```text
GET http://localhost:8080/health
GET http://localhost:8080/stellar/status
GET http://localhost:8080/metrics
```

Run the included HTTP client example:

```bash
go run ./examples/http/client/simple
```

Run the included gRPC example:

```bash
go run ./examples/grpc/server/simple
```

Run the included gRPC custom router example:

```bash
go run ./examples/grpc/server/custom-router
```

Run the interceptor examples:

```bash
go run ./examples/http/server/interceptor
go run ./examples/http/client/interceptor
go run ./examples/grpc/server/interceptor
go run ./examples/grpc/client/interceptor
```

Run the registry register example:

```bash
go run ./examples/registry/register
```

Start StellMap on `localhost:18090` first, or switch `registry.adapter` and `registry.endpoints` in `examples/registry/register/application.yml` to your Etcd, Consul, or Nacos instance.

Run the discovery example:

```bash
go run ./examples/discovery/simple
```

Then trigger a client-side discovery call:

```text
GET http://localhost:18092/api/v1/discovery/call
```

## Transport Adapters

Stellar keeps HTTP and RPC behind adapter interfaces.

Interceptor ordering for HTTP/gRPC server and client pipelines is documented in [docs/Interceptor.md](docs/Interceptor.md).

| Layer | Default | Optional implementations |
| --- | --- | --- |
| HTTP | Gin | Hertz, Chi |
| RPC | gRPC-Go | Other RPC adapters can be added later |

HTTP applications can switch adapters without changing business handlers:

```go
app := stellar.New(cfg, stellar.WithHTTPServer(":8080")) // default Gin
```

HTTP server and HTTP client use separate configuration sections:

```yaml
http:
  server:
    enabled: true
    port: 8080
    adapter: gin
    observability:
      trace: true
      metrics: true
      logs: true
  client:
    enabled: true
    timeout: 3s
    max_idle_conns: 100
    max_idle_conns_per_host: 10
    idle_conn_timeout: 90s
    observability:
      trace: true
      metrics: true
      logs: false
    clients:
      user-service:
        base_url: http://localhost:8081
        timeout: 2s
      order-service:
        base_url: http://localhost:8082
        timeout: 5s
```

RPC applications use the same lifecycle model:

```go
app := stellar.New(cfg, stellar.WithRPCServer(":9090")) // default gRPC-Go
```

gRPC server and gRPC client also use separate configuration sections. Only `grpc.server` starts a listener; `grpc.client` only configures outbound client connections:

```yaml
grpc:
  server:
    enabled: true
    port: 9090
    adapter: grpc-go
    observability:
      trace: true
      metrics: true
      logs: true
  client:
    enabled: true
    timeout: 3s
    insecure: true
    observability:
      trace: true
      metrics: true
      logs: false
    clients:
      user-service:
        target: dns:///localhost:9091
        timeout: 2s
      order-service:
        target: dns:///localhost:9092
        timeout: 5s
```

## Service Registry

Stellar exposes service discovery behind a registry adapter. The default adapter is StellMap. Etcd, Consul, and Nacos can be selected from `application.yml` without changing business code.

The service registration and discovery architecture is documented in [docs/registry-discovery-architecture.md](docs/registry-discovery-architecture.md).

If only `registry.enabled`, `adapter`, and connection fields are configured, Stellar creates a registry client and exposes it through `app.ServiceRegistry()`. If `service`, `instance_id`, and `service_endpoints` are also configured, Stellar automatically registers the current instance after server transports start and deregisters it during shutdown.

```yaml
registry:
  enabled: true
  adapter: stellmap # stellmap, etcd, consul, nacos
  endpoints:
    - http://localhost:18090
  namespace: default
  service: example-service
  instance_id: example-service-1
  zone: local
  ttl: 30s
  heartbeat_interval: 10s
  labels:
    version: v1
  metadata:
    owner: platform
  service_endpoints:
    - name: http
      protocol: http
      host: 127.0.0.1
      port: 8080
```

Switch the implementation by changing `adapter` and endpoints:

```yaml
registry:
  enabled: true
  adapter: consul
  endpoints:
    - http://localhost:8500
```

Programmatic discovery uses the same abstraction:

```go
registry, ok := app.ServiceRegistry()
instances, err := registry.Discover(ctx, stellar.ServiceQuery{
	Namespace: "default",
	Service:   "user-service",
})
```

For normal outbound calls, prefer client-side discovery. HTTP and gRPC named clients can discover endpoints through the configured registry backend and keep a local cache:

```yaml
http:
  client:
    clients:
      user-service:
        discovery:
          enabled: true
          service: user-service
          protocol: http
          endpoint_name: http
          load_balance: round_robin

grpc:
  client:
    clients:
      user-service:
        discovery:
          enabled: true
          service: user-service
          protocol: grpc
          endpoint_name: grpc
          load_balance: round_robin
```

If a named client does not configure `discovery` and does not configure a static `base_url` or `target`, Stellar inherits the top-level `discovery` config, then falls back to the global `registry` connection config.

## Data Clients

Stellar can create standard Redis, MySQL, PostgreSQL, and local cache clients from `application.yml`.

Redis uses `github.com/redis/go-redis/v9`; MySQL and PostgreSQL use the standard `database/sql` API with `github.com/go-sql-driver/mysql` and `github.com/jackc/pgx/v5/stdlib`.
Local cache is exposed through Stellar's cache adapter abstraction. BigCache is the default implementation; FreeCache can be selected with `adapter: freecache`.

```yaml
redis:
  enabled: true
  addr: localhost:6379
  db: 0
  pool_size: 16
  debug_api:
    enabled: true
    prefix: /redis
  observability:
    trace: true
    metrics: true
    logs: true

mysql:
  enabled: true
  dsn: user:password@tcp(localhost:3306)/example?parseTime=true
  max_open_conns: 25
  max_idle_conns: 5
  conn_max_lifetime: 30m
  conn_max_idle_time: 5m
  ping_on_startup: false
  debug_api:
    enabled: true
    prefix: /mysql
  observability:
    trace: true
    metrics: true
    logs: true

postgresql:
  enabled: true
  dsn: postgres://user:password@localhost:5432/example?sslmode=disable
  max_open_conns: 25
  max_idle_conns: 5
  conn_max_lifetime: 30m
  conn_max_idle_time: 5m
  ping_on_startup: false
  debug_api:
    enabled: true
    prefix: /postgresql
  observability:
    trace: true
    metrics: true
    logs: true

cache:
  enabled: true
  adapter: bigcache
  ttl: 10m
  clean_window: 1m
  shards: 1024
  hard_max_cache_size_mb: 64
  size_bytes: 67108864
  debug_api:
    enabled: true
    prefix: /cache
  observability:
    trace: true
    metrics: true
    logs: true
```

When using the programmatic API:

```go
redisClient, ok := app.RedisClient()
mysqlDB, ok := app.MySQLDB()
postgresqlDB, ok := app.PostgreSQLDB()
localCache, ok := app.Cache()
```

Switch the local cache implementation without changing application code:

```yaml
cache:
  enabled: true
  adapter: freecache
  ttl: 10m
  size_bytes: 67108864
  observability:
    metrics: true
```

Cache operations use the same framework abstraction:

```go
_ = localCache.SetString(ctx, "demo", "hello")
value, ok, err := localCache.GetString(ctx, "demo")
deleted, err := localCache.Delete(ctx, "demo")
```

Run the standalone data client examples:

```bash
go run ./examples/redis/crud-http
go run ./examples/mysql/crud-http
go run ./examples/postgresql/crud-http
go run ./examples/cache/crud-http
```

These examples use only:

```go
stellar.Run()
```

Redis/MySQL/PostgreSQL/cache clients and their debug APIs are enabled by `application.yml`, not by passing explicit starters in `main`.

Redis example API:

```text
POST   http://localhost:18081/redis/items
GET    http://localhost:18081/redis/items?key=demo
PUT    http://localhost:18081/redis/items
DELETE http://localhost:18081/redis/items?key=demo
GET    http://localhost:18081/redis/keys?pattern=*&limit=20
```

Redis create/update body:

```json
{
  "key": "demo",
  "value": "hello",
  "ttl": "5m"
}
```

MySQL example API:

```text
POST   http://localhost:18082/mysql/items
GET    http://localhost:18082/mysql/items?id=1
PUT    http://localhost:18082/mysql/items
DELETE http://localhost:18082/mysql/items?id=1
GET    http://localhost:18082/mysql/items/list?limit=20
```

MySQL create/update body:

```json
{
  "id": 1,
  "name": "demo",
  "value": "hello"
}
```

PostgreSQL example API:

```text
POST   http://localhost:18083/postgresql/items
GET    http://localhost:18083/postgresql/items?id=1
PUT    http://localhost:18083/postgresql/items
DELETE http://localhost:18083/postgresql/items?id=1
GET    http://localhost:18083/postgresql/items/list?limit=20
```

PostgreSQL create/update body:

```json
{
  "id": 1,
  "name": "demo",
  "value": "hello"
}
```

Cache example API:

```text
POST   http://localhost:18084/cache/items
GET    http://localhost:18084/cache/items?key=demo
PUT    http://localhost:18084/cache/items
DELETE http://localhost:18084/cache/items?key=demo
GET    http://localhost:18084/cache/stats
```

Cache create/update body:

```json
{
  "key": "demo",
  "value": "hello"
}
```

## OpenTelemetry

Stellar instruments HTTP server, gRPC server, HTTP client, gRPC client, Redis client, MySQL client, PostgreSQL client, and local cache client with OpenTelemetry trace, logs, and metrics.

Stellar reads configuration in this order:

1. Command line: `--config`, `--config.file`, `--stellar.config`, `--stellar.config.file`, or `--spring.config.location`.
2. Environment variables: `STELLAR_CONFIG_FILE`, `STELLAR_CONFIG`, or `STELLAR_APPLICATION_CONFIG`.
3. Default lookup: `application.yml` or `application.yaml` from the directory that contains `main.go`, then from the current working directory.

The explicit command line or environment value can be either an `application.yml` / `application.yaml` file path or a directory that contains one of those files.

```bash
go run ./examples/http/server/simple --config ./examples/http/server/simple/application.yml
STELLAR_CONFIG_FILE=./examples/http/server/simple/application.yml go run ./examples/http/server/simple
```

When code needs the configured and started app without manually loading configuration, use:

```go
app, err := stellar.Start()
if err != nil {
	return err
}
```

Use `stellar.StartWithContext(ctx)` when the app should start with a custom context.

OpenTelemetry defaults:

- `log`: defaults to local `stdout`/`stderr`; set `log.enabled: false` with `log.output: file` for local rolling files, or set `log.enabled: true` for OTLP export to `localhost:4317`.
- `trace`: when enabled, spans are generated without export; set `trace_output: otlp` for `localhost:4317`.
- `metrics`: when enabled, exposes `/metrics` on the configured HTTP port; set `metrics_output: otlp` for `localhost:4317`.

Example with explicit OTLP output:

```yaml
opentelemetry:
  log:
    enabled: true
    endpoint: localhost:4317
  trace: true
  metrics: true
  endpoint: localhost:4317
  trace_output: otlp
  metrics_output: otlp
```

Example with local rolling files:

```yaml
opentelemetry:
  log:
    enabled: false
    output: file
    dir: logs
    file_name: app.log
    max_size_bytes: 104857600
    max_backups: 5
```

When using the programmatic API, create an instrumented HTTP client:

```go
client, baseURL, err := app.NewHTTPClient("user-service")
```

When using the programmatic API, create an instrumented gRPC-Go client:

```go
conn, _, err := app.NewGRPCClient(context.Background(), "user-service")
```

## Configuration Model

| Field | Required | Description |
| --- | --- | --- |
| AppName | Yes | Logical application name |
| Environment | Yes | Runtime environment, such as `dev`, `uat`, `pre`, or `prod` |
| Zone | No | Availability zone or logical deployment zone |
| Disabled | No | Whether framework modules should be skipped during startup |

## Architecture

Detailed architecture documents:

- [Service registration and discovery architecture](docs/registry-discovery-architecture.md)
- [Interceptor ordering model](docs/Interceptor.md)

```mermaid
flowchart LR
    App[Go Service] --> Stellar[Stellar]
    Stellar --> Lifecycle[Module Lifecycle]
    Stellar --> Runtime[Runtime Metadata]
    Lifecycle --> Middleware[Stell Middleware Standards]
    Runtime --> ControlPlane[StellCloud / Control Plane]
```

## Development

Run tests:

```bash
go test ./...
```

Format code:

```bash
gofmt -w .
```

## Compatibility

Stellar follows semantic versioning once the public API stabilizes:

- `MAJOR`: incompatible API or runtime behavior changes.
- `MINOR`: backward-compatible modules, standards, or APIs.
- `PATCH`: backward-compatible fixes.

## Contribution Guidelines

- New middleware integrations should be exposed as explicit modules.
- Public API changes must describe compatibility impact.
- Framework code should prefer the Go standard library unless an external dependency provides clear value.
- Context propagation is required for startup, shutdown, client calls, and background tasks.

## License

The license will be defined before the first stable release.
