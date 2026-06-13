package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadApplicationYMLConfig(t *testing.T) {
	path := filepath.Join(t.TempDir(), "application.yml")
	content := `
app:
  name: order-service
  env: uat
  zone: zone-a
http:
  server:
    enabled: true
    port: 18080
    adapter: chi
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
    port: 19090
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
        target: dns:///localhost:19091
        timeout: 2s
      order-service:
        target: dns:///localhost:19092
        timeout: 5s
redis:
  enabled: true
  addr: localhost:6379
  db: 1
  pool_size: 16
  dial_timeout: 2s
  read_timeout: 1s
  write_timeout: 1s
  debug_api:
    enabled: true
    prefix: /redis
  observability:
    trace: true
    metrics: true
    logs: true
mysql:
  enabled: true
  dsn: user:pass@tcp(localhost:3306)/orders?parseTime=true
  max_open_conns: 25
  max_idle_conns: 5
  conn_max_lifetime: 30m
  conn_max_idle_time: 5m
  debug_api:
    enabled: true
    prefix: /mysql
  observability:
    trace: true
    metrics: true
    logs: true
postgresql:
  enabled: true
  dsn: postgres://user:pass@localhost:5432/orders?sslmode=disable
  max_open_conns: 25
  max_idle_conns: 5
  conn_max_lifetime: 30m
  conn_max_idle_time: 5m
  debug_api:
    enabled: true
    prefix: /postgresql
  observability:
    trace: true
    metrics: true
    logs: true
cache:
  enabled: true
  adapter: freecache
  ttl: 5m
  clean_window: 30s
  size_bytes: 1048576
  debug_api:
    enabled: true
    prefix: /cache
  observability:
    trace: false
    metrics: true
    logs: false
opentelemetry:
  log: true
  trace: true
  metrics: true
  trace_output: none
  metrics_output: prometheus
`
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}

	cfg, err := LoadFile(path)
	if err != nil {
		t.Fatalf("load config: %v", err)
	}
	if cfg.AppName != "order-service" {
		t.Fatalf("unexpected app name %q", cfg.AppName)
	}
	if cfg.Environment != EnvUAT {
		t.Fatalf("unexpected env %q", cfg.Environment)
	}
	if cfg.HTTP.Server == nil || cfg.HTTP.Server.Addr != ":18080" {
		t.Fatalf("unexpected http server config %#v", cfg.HTTP.Server)
	}
	if cfg.HTTP.Server.Adapter != "chi" {
		t.Fatalf("unexpected http adapter %q", cfg.HTTP.Server.Adapter)
	}
	if cfg.HTTP.Server.Observability.Trace == nil || !*cfg.HTTP.Server.Observability.Trace {
		t.Fatalf("expected http server trace observability")
	}
	if cfg.HTTP.Client == nil || cfg.HTTP.Client.Timeout != "3s" {
		t.Fatalf("unexpected http client config %#v", cfg.HTTP.Client)
	}
	if cfg.HTTP.Client.MaxIdleConns != 100 {
		t.Fatalf("unexpected max idle conns %d", cfg.HTTP.Client.MaxIdleConns)
	}
	if cfg.HTTP.Client.Observability.Logs == nil || *cfg.HTTP.Client.Observability.Logs {
		t.Fatalf("expected http client logs observability disabled")
	}
	userClient := cfg.HTTP.Client.Clients["user-service"]
	if userClient.BaseURL != "http://localhost:8081" || userClient.Timeout != "2s" {
		t.Fatalf("unexpected user-service client config %#v", userClient)
	}
	if cfg.GRPC.Server == nil || cfg.GRPC.Server.Addr != ":19090" {
		t.Fatalf("unexpected grpc server config %#v", cfg.GRPC.Server)
	}
	if cfg.GRPC.Server.Adapter != "grpc-go" {
		t.Fatalf("unexpected grpc adapter %q", cfg.GRPC.Server.Adapter)
	}
	if cfg.GRPC.Server.Observability.Trace == nil || !*cfg.GRPC.Server.Observability.Trace {
		t.Fatalf("expected grpc server trace observability")
	}
	if cfg.GRPC.Client == nil || cfg.GRPC.Client.Timeout != "3s" {
		t.Fatalf("unexpected grpc client config %#v", cfg.GRPC.Client)
	}
	if cfg.GRPC.Client.Insecure == nil || !*cfg.GRPC.Client.Insecure {
		t.Fatalf("expected grpc client insecure")
	}
	if cfg.GRPC.Client.Observability.Logs == nil || *cfg.GRPC.Client.Observability.Logs {
		t.Fatalf("expected grpc client logs observability disabled")
	}
	grpcUserClient := cfg.GRPC.Client.Clients["user-service"]
	if grpcUserClient.Target != "dns:///localhost:19091" || grpcUserClient.Timeout != "2s" {
		t.Fatalf("unexpected grpc user-service client config %#v", grpcUserClient)
	}
	if cfg.Redis == nil || cfg.Redis.Addr != "localhost:6379" || cfg.Redis.DB != 1 {
		t.Fatalf("unexpected redis config %#v", cfg.Redis)
	}
	if cfg.Redis.DebugAPI == nil || cfg.Redis.DebugAPI.Enabled == nil || !*cfg.Redis.DebugAPI.Enabled || cfg.Redis.DebugAPI.Prefix != "/redis" {
		t.Fatalf("unexpected redis debug api config %#v", cfg.Redis.DebugAPI)
	}
	if cfg.Redis.Observability.Logs == nil || !*cfg.Redis.Observability.Logs {
		t.Fatalf("expected redis logs observability")
	}
	if cfg.MySQL == nil || cfg.MySQL.Driver != "mysql" || cfg.MySQL.MaxOpenConns != 25 {
		t.Fatalf("unexpected mysql config %#v", cfg.MySQL)
	}
	if cfg.MySQL.DebugAPI == nil || cfg.MySQL.DebugAPI.Enabled == nil || !*cfg.MySQL.DebugAPI.Enabled || cfg.MySQL.DebugAPI.Prefix != "/mysql" {
		t.Fatalf("unexpected mysql debug api config %#v", cfg.MySQL.DebugAPI)
	}
	if cfg.MySQL.Observability.Metrics == nil || !*cfg.MySQL.Observability.Metrics {
		t.Fatalf("expected mysql metrics observability")
	}
	if cfg.PostgreSQL == nil || cfg.PostgreSQL.Driver != "pgx" || cfg.PostgreSQL.MaxOpenConns != 25 {
		t.Fatalf("unexpected postgresql config %#v", cfg.PostgreSQL)
	}
	if cfg.PostgreSQL.DebugAPI == nil || cfg.PostgreSQL.DebugAPI.Enabled == nil || !*cfg.PostgreSQL.DebugAPI.Enabled || cfg.PostgreSQL.DebugAPI.Prefix != "/postgresql" {
		t.Fatalf("unexpected postgresql debug api config %#v", cfg.PostgreSQL.DebugAPI)
	}
	if cfg.PostgreSQL.Observability.Trace == nil || !*cfg.PostgreSQL.Observability.Trace {
		t.Fatalf("expected postgresql trace observability")
	}
	if cfg.Cache == nil || cfg.Cache.Adapter != "freecache" || cfg.Cache.TTL != "5m" || cfg.Cache.SizeBytes != 1048576 {
		t.Fatalf("unexpected cache config %#v", cfg.Cache)
	}
	if cfg.Cache.DebugAPI == nil || cfg.Cache.DebugAPI.Enabled == nil || !*cfg.Cache.DebugAPI.Enabled || cfg.Cache.DebugAPI.Prefix != "/cache" {
		t.Fatalf("unexpected cache debug api config %#v", cfg.Cache.DebugAPI)
	}
	if cfg.Cache.Observability.Metrics == nil || !*cfg.Cache.Observability.Metrics {
		t.Fatalf("expected cache metrics observability")
	}
	if cfg.Cache.Observability.Logs == nil || *cfg.Cache.Observability.Logs {
		t.Fatalf("expected cache logs observability disabled")
	}
	if cfg.Starter.OpenTelemetry == nil || !cfg.Starter.OpenTelemetry.Log.Enabled {
		t.Fatalf("expected opentelemetry log starter")
	}
}

func TestLoadApplicationYAMLConfig(t *testing.T) {
	path := filepath.Join(t.TempDir(), "application.yaml")
	content := `
app:
  name: user-service
  env: dev
http:
  server:
    port: 8081
  client:
    timeout: 3s
    clients:
      user-service:
        base_url: http://localhost:8081
        timeout: 2s
opentelemetry:
  log:
    enabled: false
    output: file
    dir: logs
    file_name: app.log
    max_size_bytes: 104857600
    max_backups: 5
`
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}

	cfg, err := LoadFile(path)
	if err != nil {
		t.Fatalf("load config: %v", err)
	}
	if cfg.AppName != "user-service" {
		t.Fatalf("unexpected app name %q", cfg.AppName)
	}
	if cfg.HTTP.Server == nil || cfg.HTTP.Server.Addr != ":8081" {
		t.Fatalf("unexpected http server config %#v", cfg.HTTP.Server)
	}
	if cfg.HTTP.Client == nil || cfg.HTTP.Client.Clients["user-service"].BaseURL != "http://localhost:8081" {
		t.Fatalf("unexpected http client config %#v", cfg.HTTP.Client)
	}
	if cfg.Starter.OpenTelemetry == nil || cfg.Starter.OpenTelemetry.Log.Output != "file" {
		t.Fatalf("unexpected opentelemetry config: %#v", cfg.Starter.OpenTelemetry)
	}
	if cfg.Starter.OpenTelemetry.Log.FileName != "app.log" {
		t.Fatalf("unexpected log file name %q", cfg.Starter.OpenTelemetry.Log.FileName)
	}
}

func TestLoadUsesCommandLineConfigPath(t *testing.T) {
	path := writeTestConfig(t, filepath.Join(t.TempDir(), "application.yaml"), "cli-service")
	setArgs(t, "stellar", "--config", path)
	clearConfigEnv(t)

	cfg, err := Load()
	if err != nil {
		t.Fatalf("load config: %v", err)
	}
	if cfg.AppName != "cli-service" {
		t.Fatalf("unexpected app name %q", cfg.AppName)
	}
}

func TestLoadUsesCommandLineConfigDirectory(t *testing.T) {
	dir := t.TempDir()
	writeTestConfig(t, filepath.Join(dir, "application.yml"), "cli-dir-service")
	setArgs(t, "stellar", "--config.file="+dir)
	clearConfigEnv(t)

	cfg, err := Load()
	if err != nil {
		t.Fatalf("load config: %v", err)
	}
	if cfg.AppName != "cli-dir-service" {
		t.Fatalf("unexpected app name %q", cfg.AppName)
	}
}

func TestLoadUsesEnvConfigPath(t *testing.T) {
	path := writeTestConfig(t, filepath.Join(t.TempDir(), "application.yaml"), "env-service")
	setArgs(t, "stellar")
	clearConfigEnv(t)
	t.Setenv(EnvConfigFile, path)

	cfg, err := Load()
	if err != nil {
		t.Fatalf("load config: %v", err)
	}
	if cfg.AppName != "env-service" {
		t.Fatalf("unexpected app name %q", cfg.AppName)
	}
}

func TestLoadCommandLineConfigPathOverridesEnv(t *testing.T) {
	envPath := writeTestConfig(t, filepath.Join(t.TempDir(), "application.yaml"), "env-service")
	cliPath := writeTestConfig(t, filepath.Join(t.TempDir(), "application.yaml"), "cli-service")
	setArgs(t, "stellar", "--stellar.config", cliPath)
	clearConfigEnv(t)
	t.Setenv(EnvConfigFile, envPath)

	cfg, err := Load()
	if err != nil {
		t.Fatalf("load config: %v", err)
	}
	if cfg.AppName != "cli-service" {
		t.Fatalf("unexpected app name %q", cfg.AppName)
	}
}

func TestLoadRejectsMissingCommandLineConfigValue(t *testing.T) {
	setArgs(t, "stellar", "--config")
	clearConfigEnv(t)

	if _, err := Load(); err == nil {
		t.Fatalf("expected missing command line config value to be rejected")
	}
}

func TestLoadFileRejectsUnsupportedConfigName(t *testing.T) {
	path := filepath.Join(t.TempDir(), "stellar.yaml")
	if err := os.WriteFile(path, []byte("app:\n  name: order-service\n"), 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}

	if _, err := LoadFile(path); err == nil {
		t.Fatalf("expected unsupported config file name to be rejected")
	}
}

func TestConfigPathsPreferMainDirBeforeWorkingDir(t *testing.T) {
	mainDir := filepath.Join("repo", "examples", "http", "server", "simple")
	workingDir := filepath.Join("repo")

	paths := configPathsInDirs(mainDir, workingDir)

	if paths[0] != filepath.Join(mainDir, "application.yml") {
		t.Fatalf("expected main dir application.yml first, got %q", paths[0])
	}
	if paths[1] != filepath.Join(mainDir, "application.yaml") {
		t.Fatalf("expected main dir application.yaml second, got %q", paths[1])
	}
	if paths[2] != filepath.Join(workingDir, "application.yml") {
		t.Fatalf("expected working dir after main dir candidates, got %q", paths[2])
	}
}

func writeTestConfig(t *testing.T, path string, appName string) string {
	t.Helper()
	content := "app:\n  name: " + appName + "\n"
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}
	return path
}

func setArgs(t *testing.T, args ...string) {
	t.Helper()
	originalArgs := os.Args
	os.Args = args
	t.Cleanup(func() {
		os.Args = originalArgs
	})
}

func clearConfigEnv(t *testing.T) {
	t.Helper()
	t.Setenv(EnvConfigFile, "")
	t.Setenv(EnvConfig, "")
	t.Setenv(EnvApplicationConfig, "")
}
