<p align="right">
<a href="README.md">English</a> | 简体中文
</p>

# Stellar

`stellar` 是 Stell 中间件生态的 Go 框架。它为需要接入 Stell 配置、服务发现、消息、可观测性、治理和平台运维标准的服务提供统一的应用基础。

它与 [`stellhub/stellflux`](https://github.com/stellhub/stellflux) 定位一致，同时遵循 Go 的工程习惯：小包、显式组合、上下文传递、优先使用标准库，以及可预测的生命周期管理。

## 定位

Stellar 不是一个中间件服务器，也不承载业务逻辑。它是一个框架层，面向需要统一接入 Stell 中间件能力的 Go 服务。

## 核心职责

- 提供统一的应用配置和运行时元数据。
- 定义轻量级的 Stell 中间件模块生命周期。
- 标准化 Go 服务接入 StellMap、StellFlow、StellNula、StellSpec、StellOrbit、StellGate 和 StellAtlas 的方式。
- 暴露框架状态，服务于健康检查、控制面和平台控制台。
- 保持可观测性、服务身份、环境和可用区元数据一致。

## 中间件标准

| 标准 | 职责 |
| --- | --- |
| StellMap | 服务发现与注册中心集成 |
| StellFlow | 消息与事件流集成 |
| StellNula | 配置中心集成 |
| StellSpec | 可观测性与日志查询标准集成 |
| StellOrbit | 流量治理、路由、重试和生命周期策略集成 |
| StellGate | API 网关与入口标准集成 |
| StellAtlas | CMDB、资产清单、拓扑和生命周期元数据集成 |

## 当前状态

| 项目 | 值 |
| --- | --- |
| 稳定性 | 早期开发 |
| 语言 | Go |
| 项目类型 | Go 框架 |
| 目标用户 | Go 微服务、平台服务、基础设施组件 |
| 维护者 | StellHub |

## 快速开始

安装模块：

```bash
go get github.com/stellhub/stellar
```

创建应用：

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

添加 `application.yml`：

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

运行 HTTP server 示例：

```bash
go run ./examples/http/server/simple
```

然后访问：

```text
GET http://localhost:8080/health
GET http://localhost:8080/stellar/status
GET http://localhost:8080/metrics
```

运行 HTTP client 示例：

```bash
go run ./examples/http/client/simple
```

运行 gRPC 示例：

```bash
go run ./examples/grpc/server/simple
```

运行 gRPC custom router 示例：

```bash
go run ./examples/grpc/server/custom-router
```

运行 interceptor 示例：

```bash
go run ./examples/http/server/interceptor
go run ./examples/http/client/interceptor
go run ./examples/grpc/server/interceptor
go run ./examples/grpc/client/interceptor
```

运行注册中心 register 示例：

```bash
go run ./examples/registry/register
```

需要先在 `localhost:18090` 启动 StellMap，或者将 `examples/registry/register/application.yml` 中的 `registry.adapter` 和 `registry.endpoints` 切换为你的 Etcd、Consul 或 Nacos 实例。

运行 discovery 示例：

```bash
go run ./examples/discovery/simple
```

然后触发一次客户端侧服务发现调用：

```text
GET http://localhost:18092/api/v1/discovery/call
```

## 传输适配器

Stellar 将 HTTP 和 RPC 隔离在适配器接口之后。

HTTP/gRPC server 与 client 的拦截器顺序规范见 [docs/Interceptor.md](docs/Interceptor.md)。

| 层 | 默认实现 | 可选实现 |
| --- | --- | --- |
| HTTP | Gin | Hertz、Chi |
| RPC | gRPC-Go | 后续可扩展其它 RPC 适配器 |

HTTP 应用可以在不修改业务 handler 的情况下切换适配器：

```go
app := stellar.New(cfg, stellar.WithHTTPServer(":8080")) // 默认 Gin
```

HTTP server 和 HTTP client 使用独立配置：

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

RPC 应用使用同样的生命周期模型：

```go
app := stellar.New(cfg, stellar.WithRPCServer(":9090")) // 默认 gRPC-Go
```

gRPC server 和 gRPC client 也使用独立配置。只有 `grpc.server` 会启动监听器；`grpc.client` 只配置出站客户端连接：

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

## 服务注册中心

Stellar 将服务发现隔离在 registry adapter 抽象之后。默认实现是 StellMap，也可以在 `application.yml` 中选择 Etcd、Consul 或 Nacos，不需要修改业务代码。

服务注册与服务发现架构设计见 [docs/registry-discovery-architecture.md](docs/registry-discovery-architecture.md)。

如果只配置了 `registry.enabled`、`adapter` 和连接信息，Stellar 会创建注册中心客户端，并通过 `app.ServiceRegistry()` 暴露。如果同时配置了 `service`、`instance_id` 和 `service_endpoints`，Stellar 会在服务端 transport 启动后自动注册当前实例，并在应用停止时注销。

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

切换实现只需要修改 `adapter` 和 endpoints：

```yaml
registry:
  enabled: true
  adapter: consul
  endpoints:
    - http://localhost:8500
```

程序化服务发现使用同一套抽象：

```go
registry, ok := app.ServiceRegistry()
instances, err := registry.Discover(ctx, stellar.ServiceQuery{
	Namespace: "default",
	Service:   "user-service",
})
```

正常出站调用建议使用客户端侧 discovery。HTTP 和 gRPC named client 可以通过配置的注册中心后端发现 endpoint，并维护本地缓存：

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

如果 named client 没有配置 `discovery`，也没有配置静态 `base_url` 或 `target`，Stellar 会先继承顶层 `discovery` 配置，再回退到全局 `registry` 的连接配置。

## 数据客户端

Stellar 可以根据 `application.yml` 创建标准 Redis、MySQL、PostgreSQL 和本地缓存客户端。

Redis 使用 `github.com/redis/go-redis/v9`；MySQL 和 PostgreSQL 使用标准 `database/sql` API，并分别接入 `github.com/go-sql-driver/mysql` 与 `github.com/jackc/pgx/v5/stdlib`。
本地缓存通过 Stellar 的 cache adapter 抽象暴露。BigCache 是默认实现；可以通过 `adapter: freecache` 选择 FreeCache。

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

使用程序化 API 时：

```go
redisClient, ok := app.RedisClient()
mysqlDB, ok := app.MySQLDB()
postgresqlDB, ok := app.PostgreSQLDB()
localCache, ok := app.Cache()
```

不修改应用代码即可切换本地缓存实现：

```yaml
cache:
  enabled: true
  adapter: freecache
  ttl: 10m
  size_bytes: 67108864
  observability:
    metrics: true
```

缓存操作使用同一套框架抽象：

```go
_ = localCache.SetString(ctx, "demo", "hello")
value, ok, err := localCache.GetString(ctx, "demo")
deleted, err := localCache.Delete(ctx, "demo")
```

运行独立数据客户端示例：

```bash
go run ./examples/redis/crud-http
go run ./examples/mysql/crud-http
go run ./examples/postgresql/crud-http
go run ./examples/cache/crud-http
```

这些示例只需要：

```go
stellar.Run()
```

Redis、MySQL、PostgreSQL、cache 客户端及其 debug API 都由 `application.yml` 启用，不需要在 `main` 中显式传入 starter。

Redis 示例 API：

```text
POST   http://localhost:18081/redis/items
GET    http://localhost:18081/redis/items?key=demo
PUT    http://localhost:18081/redis/items
DELETE http://localhost:18081/redis/items?key=demo
GET    http://localhost:18081/redis/keys?pattern=*&limit=20
```

Redis 创建/更新请求体：

```json
{
  "key": "demo",
  "value": "hello",
  "ttl": "5m"
}
```

MySQL 示例 API：

```text
POST   http://localhost:18082/mysql/items
GET    http://localhost:18082/mysql/items?id=1
PUT    http://localhost:18082/mysql/items
DELETE http://localhost:18082/mysql/items?id=1
GET    http://localhost:18082/mysql/items/list?limit=20
```

MySQL 创建/更新请求体：

```json
{
  "id": 1,
  "name": "demo",
  "value": "hello"
}
```

PostgreSQL 示例 API：

```text
POST   http://localhost:18083/postgresql/items
GET    http://localhost:18083/postgresql/items?id=1
PUT    http://localhost:18083/postgresql/items
DELETE http://localhost:18083/postgresql/items?id=1
GET    http://localhost:18083/postgresql/items/list?limit=20
```

PostgreSQL 创建/更新请求体：

```json
{
  "id": 1,
  "name": "demo",
  "value": "hello"
}
```

Cache 示例 API：

```text
POST   http://localhost:18084/cache/items
GET    http://localhost:18084/cache/items?key=demo
PUT    http://localhost:18084/cache/items
DELETE http://localhost:18084/cache/items?key=demo
GET    http://localhost:18084/cache/stats
```

Cache 创建/更新请求体：

```json
{
  "key": "demo",
  "value": "hello"
}
```

## OpenTelemetry

Stellar 会为 HTTP server、gRPC server、HTTP client、gRPC client、Redis client、MySQL client、PostgreSQL client 和本地 cache client 接入 OpenTelemetry trace、logs 与 metrics。

Stellar 会按下面顺序读取配置：

1. 命令行参数：`--config`、`--config.file`、`--stellar.config`、`--stellar.config.file` 或 `--spring.config.location`。
2. 环境变量：`STELLAR_CONFIG_FILE`、`STELLAR_CONFIG` 或 `STELLAR_APPLICATION_CONFIG`。
3. 默认查找：先从包含 `main.go` 的目录读取 `application.yml` 或 `application.yaml`，再从当前工作目录读取。

命令行或环境变量显式指定的值可以是 `application.yml` / `application.yaml` 文件路径，也可以是包含这些文件的目录。

```bash
go run ./examples/http/server/simple --config ./examples/http/server/simple/application.yml
STELLAR_CONFIG_FILE=./examples/http/server/simple/application.yml go run ./examples/http/server/simple
```

如果代码需要拿到已配置并已启动的 app，但不想手动加载配置，可以使用：

```go
app, err := stellar.Start()
if err != nil {
	return err
}
```

当应用需要使用自定义 context 启动时，使用 `stellar.StartWithContext(ctx)`。

OpenTelemetry 默认行为：

- `log`：默认输出到本地 `stdout`/`stderr`；设置 `log.enabled: false` 并配合 `log.output: file` 可输出到本地滚动文件；设置 `log.enabled: true` 可通过 OTLP 输出到 `localhost:4317`。
- `trace`：启用后会生成 spans，但默认不导出；设置 `trace_output: otlp` 可输出到 `localhost:4317`。
- `metrics`：启用后会在当前 HTTP 端口暴露 `/metrics`；设置 `metrics_output: otlp` 可输出到 `localhost:4317`。

显式 OTLP 输出示例：

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

本地滚动文件示例：

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

使用程序化 API 创建已接入观测能力的 HTTP client：

```go
client, baseURL, err := app.NewHTTPClient("user-service")
```

使用程序化 API 创建已接入观测能力的 gRPC-Go client：

```go
conn, _, err := app.NewGRPCClient(context.Background(), "user-service")
```

## 配置模型

| 字段 | 是否必需 | 说明 |
| --- | --- | --- |
| AppName | 是 | 应用逻辑名称 |
| Environment | 是 | 运行环境，例如 `dev`、`uat`、`pre` 或 `prod` |
| Zone | 否 | 可用区或逻辑部署区域 |
| Disabled | 否 | 是否跳过框架模块启动 |

## 架构

详细架构文档：

- [服务注册与服务发现架构设计](docs/registry-discovery-architecture.md)
- [拦截器顺序模型](docs/Interceptor.md)

```mermaid
flowchart LR
    App[Go Service] --> Stellar[Stellar]
    Stellar --> Lifecycle[Module Lifecycle]
    Stellar --> Runtime[Runtime Metadata]
    Lifecycle --> Middleware[Stell Middleware Standards]
    Runtime --> ControlPlane[StellCloud / Control Plane]
```

## 开发

运行测试：

```bash
go test ./...
```

格式化代码：

```bash
gofmt -w .
```

## 兼容性

Stellar 在公共 API 稳定后会遵循语义化版本：

- `MAJOR`：不兼容的 API 或运行时行为变更。
- `MINOR`：向后兼容的模块、标准或 API。
- `PATCH`：向后兼容的修复。

## 贡献指南

- 新的中间件集成应以显式模块暴露。
- 公共 API 变更必须说明兼容性影响。
- 框架代码应优先使用 Go 标准库，除非外部依赖能带来明确价值。
- 启动、关闭、客户端调用和后台任务都必须传递 `context`。

## 许可证

许可证会在第一个稳定版本发布前确定。
