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
	mainDir := filepath.Join("repo", "examples", "http-examples")
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
