package config

import (
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"

	"github.com/goccy/go-yaml"
)

const (
	EnvDev  Environment = "dev"
	EnvUAT  Environment = "uat"
	EnvPre  Environment = "pre"
	EnvProd Environment = "prod"
)

type Environment string

type HTTPConfig struct {
	Server *HTTPServerConfig `yaml:"server"`
	Client *HTTPClientConfig `yaml:"client"`
}

type HTTPServerConfig struct {
	Enabled       *bool                     `yaml:"enabled"`
	Port          int                       `yaml:"port"`
	Addr          string                    `yaml:"addr"`
	Adapter       string                    `yaml:"adapter"`
	Observability ObservabilitySignalConfig `yaml:"observability"`
}

type HTTPClientConfig struct {
	Enabled             *bool                            `yaml:"enabled"`
	Timeout             string                           `yaml:"timeout"`
	MaxIdleConns        int                              `yaml:"max_idle_conns"`
	MaxIdleConnsPerHost int                              `yaml:"max_idle_conns_per_host"`
	IdleConnTimeout     string                           `yaml:"idle_conn_timeout"`
	Clients             map[string]HTTPNamedClientConfig `yaml:"clients"`
	Observability       ObservabilitySignalConfig        `yaml:"observability"`
}

type HTTPNamedClientConfig struct {
	BaseURL string `yaml:"base_url"`
	Timeout string `yaml:"timeout"`
}

type ObservabilitySignalConfig struct {
	Trace   *bool `yaml:"trace"`
	Metrics *bool `yaml:"metrics"`
	Logs    *bool `yaml:"logs"`
}

type GRPCConfig struct {
	Server *GRPCServerConfig `yaml:"server"`
	Client *GRPCClientConfig `yaml:"client"`
}

type GRPCServerConfig struct {
	Enabled       *bool                     `yaml:"enabled"`
	Port          int                       `yaml:"port"`
	Addr          string                    `yaml:"addr"`
	Adapter       string                    `yaml:"adapter"`
	Observability ObservabilitySignalConfig `yaml:"observability"`
}

type GRPCClientConfig struct {
	Enabled       *bool                            `yaml:"enabled"`
	Target        string                           `yaml:"target"`
	Timeout       string                           `yaml:"timeout"`
	Authority     string                           `yaml:"authority"`
	Insecure      *bool                            `yaml:"insecure"`
	Clients       map[string]GRPCNamedClientConfig `yaml:"clients"`
	Observability ObservabilitySignalConfig        `yaml:"observability"`
}

type GRPCNamedClientConfig struct {
	Target    string `yaml:"target"`
	Timeout   string `yaml:"timeout"`
	Authority string `yaml:"authority"`
	Insecure  *bool  `yaml:"insecure"`
}

type RedisConfig struct {
	Enabled       *bool                     `yaml:"enabled"`
	Addr          string                    `yaml:"addr"`
	Username      string                    `yaml:"username"`
	Password      string                    `yaml:"password"`
	DB            int                       `yaml:"db"`
	ClientName    string                    `yaml:"client_name"`
	Protocol      int                       `yaml:"protocol"`
	MaxRetries    int                       `yaml:"max_retries"`
	PoolSize      int                       `yaml:"pool_size"`
	MinIdleConns  int                       `yaml:"min_idle_conns"`
	DialTimeout   string                    `yaml:"dial_timeout"`
	ReadTimeout   string                    `yaml:"read_timeout"`
	WriteTimeout  string                    `yaml:"write_timeout"`
	Observability ObservabilitySignalConfig `yaml:"observability"`
	DebugAPI      *DebugAPIConfig           `yaml:"debug_api"`
}

type MySQLConfig struct {
	Enabled         *bool                     `yaml:"enabled"`
	Driver          string                    `yaml:"driver"`
	DSN             string                    `yaml:"dsn"`
	MaxOpenConns    int                       `yaml:"max_open_conns"`
	MaxIdleConns    int                       `yaml:"max_idle_conns"`
	ConnMaxLifetime string                    `yaml:"conn_max_lifetime"`
	ConnMaxIdleTime string                    `yaml:"conn_max_idle_time"`
	PingOnStartup   bool                      `yaml:"ping_on_startup"`
	PingTimeout     string                    `yaml:"ping_timeout"`
	Observability   ObservabilitySignalConfig `yaml:"observability"`
	DebugAPI        *DebugAPIConfig           `yaml:"debug_api"`
}

type DebugAPIConfig struct {
	Enabled *bool  `yaml:"enabled"`
	Prefix  string `yaml:"prefix"`
}

type Config struct {
	AppName     string
	Environment Environment
	Zone        string
	Version     string
	Disabled    bool
	HTTP        HTTPConfig
	GRPC        GRPCConfig
	Redis       *RedisConfig
	MySQL       *MySQLConfig
	Starter     StarterConfig
	Metadata    map[string]string
}

type StarterConfig struct {
	HTTP          *HTTPConfig                 `yaml:"http"`
	GRPC          *GRPCConfig                 `yaml:"grpc"`
	OpenTelemetry *OpenTelemetryStarterConfig `yaml:"opentelemetry"`
}

type HTTPStarterConfig = HTTPServerConfig

type GRPCStarterConfig = GRPCServerConfig

type OpenTelemetryStarterConfig struct {
	Log     OpenTelemetryLogConfig `yaml:"log"`
	Trace   bool                   `yaml:"trace"`
	Metrics bool                   `yaml:"metrics"`

	Endpoint string `yaml:"endpoint"`
	Insecure *bool  `yaml:"insecure"`

	TraceOutput   string `yaml:"trace_output"`
	TraceEndpoint string `yaml:"trace_endpoint"`

	MetricsOutput   string `yaml:"metrics_output"`
	MetricsEndpoint string `yaml:"metrics_endpoint"`
	MetricsPath     string `yaml:"metrics_path"`
}

type OpenTelemetryLogConfig struct {
	Enabled      bool   `yaml:"enabled"`
	Output       string `yaml:"output"`
	Format       string `yaml:"format"`
	Level        string `yaml:"level"`
	Dir          string `yaml:"dir"`
	FileName     string `yaml:"file_name"`
	MaxSizeBytes int64  `yaml:"max_size_bytes"`
	MaxBackups   int    `yaml:"max_backups"`
	Endpoint     string `yaml:"endpoint"`
}

func (c *OpenTelemetryLogConfig) UnmarshalYAML(unmarshal func(any) error) error {
	var enabled bool
	if err := unmarshal(&enabled); err == nil {
		c.Enabled = enabled
		return nil
	}

	type rawOpenTelemetryLogConfig OpenTelemetryLogConfig
	var raw rawOpenTelemetryLogConfig
	if err := unmarshal(&raw); err != nil {
		return err
	}
	*c = OpenTelemetryLogConfig(raw)
	return nil
}

type fileConfig struct {
	App           appFileConfig               `yaml:"app"`
	HTTP          *HTTPConfig                 `yaml:"http"`
	GRPC          *GRPCConfig                 `yaml:"grpc"`
	Redis         *RedisConfig                `yaml:"redis"`
	MySQL         *MySQLConfig                `yaml:"mysql"`
	OpenTelemetry *OpenTelemetryStarterConfig `yaml:"opentelemetry"`
}

type appFileConfig struct {
	Name        string            `yaml:"name"`
	Env         string            `yaml:"env"`
	Environment string            `yaml:"environment"`
	Zone        string            `yaml:"zone"`
	Version     string            `yaml:"version"`
	Disabled    bool              `yaml:"disabled"`
	Metadata    map[string]string `yaml:"metadata"`
}

var ErrConfigFileNotFound = errors.New("stellar: config file not found")

func (e Environment) Valid() bool {
	switch e {
	case EnvDev, EnvUAT, EnvPre, EnvProd:
		return true
	default:
		return false
	}
}

func (c Config) Normalize() Config {
	if c.Environment == "" {
		c.Environment = EnvDev
	}
	if c.HTTP.Server == nil && c.HTTP.Client == nil && c.Starter.HTTP != nil {
		c.HTTP = *c.Starter.HTTP
	}
	c.HTTP = c.HTTP.Normalize()
	if c.GRPC.Server == nil && c.GRPC.Client == nil && c.Starter.GRPC != nil {
		c.GRPC = *c.Starter.GRPC
	}
	c.GRPC = c.GRPC.Normalize()
	if c.Redis != nil {
		redis := *c.Redis
		if strings.TrimSpace(redis.Addr) == "" {
			redis.Addr = "localhost:6379"
		}
		c.Redis = &redis
	}
	if c.MySQL != nil {
		mysql := *c.MySQL
		if strings.TrimSpace(mysql.Driver) == "" {
			mysql.Driver = "mysql"
		}
		c.MySQL = &mysql
	}
	if c.Metadata == nil {
		c.Metadata = map[string]string{}
		return c
	}

	metadata := make(map[string]string, len(c.Metadata))
	for key, value := range c.Metadata {
		metadata[key] = value
	}
	c.Metadata = metadata
	return c
}

func (c HTTPConfig) Normalize() HTTPConfig {
	if c.Server != nil {
		server := *c.Server
		server.Addr = addrFromPort(server.Addr, server.Port, ":8080")
		c.Server = &server
	}
	if c.Client != nil {
		client := *c.Client
		if client.Clients != nil {
			clients := make(map[string]HTTPNamedClientConfig, len(client.Clients))
			for key, value := range client.Clients {
				clients[key] = value
			}
			client.Clients = clients
		}
		c.Client = &client
	}
	return c
}

func (c HTTPConfig) ServerAddr() string {
	if c.Server != nil && strings.TrimSpace(c.Server.Addr) != "" {
		return c.Server.Addr
	}
	return ":8080"
}

func (c GRPCConfig) Normalize() GRPCConfig {
	if c.Server != nil {
		server := *c.Server
		server.Addr = addrFromPort(server.Addr, server.Port, ":9090")
		c.Server = &server
	}
	if c.Client != nil {
		client := *c.Client
		if client.Clients != nil {
			clients := make(map[string]GRPCNamedClientConfig, len(client.Clients))
			for key, value := range client.Clients {
				clients[key] = value
			}
			client.Clients = clients
		}
		c.Client = &client
	}
	return c
}

func (c GRPCConfig) ServerAddr() string {
	if c.Server != nil && strings.TrimSpace(c.Server.Addr) != "" {
		return c.Server.Addr
	}
	return ":9090"
}

func Load() (Config, error) {
	for _, candidate := range DefaultConfigPaths() {
		if _, err := os.Stat(candidate); err == nil {
			return LoadFile(candidate)
		}
	}
	return Config{}, ErrConfigFileNotFound
}

func DefaultConfigPaths() []string {
	dirs := []string{}
	if mainDir := mainSourceDir(); mainDir != "" {
		dirs = append(dirs, mainDir)
	}
	if cwd, err := os.Getwd(); err == nil {
		dirs = append(dirs, cwd)
	}
	return configPathsInDirs(dirs...)
}

func configPathsInDirs(dirs ...string) []string {
	names := []string{
		"application.yml",
		"application.yaml",
	}
	seen := map[string]struct{}{}
	paths := make([]string, 0, len(dirs)*len(names))
	for _, dir := range dirs {
		if strings.TrimSpace(dir) == "" {
			continue
		}
		for _, name := range names {
			path := filepath.Join(dir, name)
			if _, ok := seen[path]; ok {
				continue
			}
			seen[path] = struct{}{}
			paths = append(paths, path)
		}
	}
	if len(paths) == 0 {
		return names
	}
	return paths
}

func mainSourceDir() string {
	pcs := make([]uintptr, 32)
	n := runtime.Callers(0, pcs)
	frames := runtime.CallersFrames(pcs[:n])
	for {
		frame, more := frames.Next()
		if frame.Function == "main.main" && frame.File != "" {
			return filepath.Dir(frame.File)
		}
		if !more {
			break
		}
	}
	return ""
}

func DefaultConfigNames() []string {
	return []string{
		"application.yml",
		"application.yaml",
	}
}

func LoadFile(path string) (Config, error) {
	if !isSupportedConfigFileName(path) {
		return Config{}, errors.New("stellar: only application.yml or application.yaml is supported")
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return Config{}, err
	}

	var raw fileConfig
	switch strings.ToLower(filepath.Ext(path)) {
	case ".yaml", ".yml":
		if err := yaml.Unmarshal(data, &raw); err != nil {
			return Config{}, err
		}
	default:
		return Config{}, errors.New("stellar: only application.yml or application.yaml is supported")
	}

	env := Environment(raw.App.Environment)
	if env == "" {
		env = Environment(raw.App.Env)
	}

	cfg := Config{
		AppName:     raw.App.Name,
		Environment: env,
		Zone:        raw.App.Zone,
		Version:     raw.App.Version,
		Disabled:    raw.App.Disabled,
		HTTP:        derefHTTPConfig(raw.HTTP),
		GRPC:        derefGRPCConfig(raw.GRPC),
		Redis:       raw.Redis,
		MySQL:       raw.MySQL,
		Starter: StarterConfig{
			HTTP:          raw.HTTP,
			GRPC:          raw.GRPC,
			OpenTelemetry: raw.OpenTelemetry,
		},
		Metadata: raw.App.Metadata,
	}
	return cfg.Normalize(), nil
}

func isSupportedConfigFileName(path string) bool {
	switch strings.ToLower(filepath.Base(path)) {
	case "application.yml", "application.yaml":
		return true
	default:
		return false
	}
}

func FromEnv(base Config) Config {
	cfg := base.Normalize()
	if value := strings.TrimSpace(os.Getenv("STELLAR_APP_NAME")); value != "" {
		cfg.AppName = value
	}
	if value := strings.TrimSpace(os.Getenv("STELLAR_ENV")); value != "" {
		cfg.Environment = Environment(value)
	}
	if value := strings.TrimSpace(os.Getenv("STELLAR_ZONE")); value != "" {
		cfg.Zone = value
	}
	if value := strings.TrimSpace(os.Getenv("STELLAR_VERSION")); value != "" {
		cfg.Version = value
	}
	if value := strings.TrimSpace(os.Getenv("STELLAR_HTTP_ADDR")); value != "" {
		ensureHTTPServerConfig(&cfg).Addr = value
	}
	if value := strings.TrimSpace(os.Getenv("STELLAR_GRPC_ADDR")); value != "" {
		ensureGRPCServerConfig(&cfg).Addr = value
	}
	if value, ok := envBool("STELLAR_DISABLED"); ok {
		cfg.Disabled = value
	}
	if value, ok := envBool("STELLAR_HTTP_ENABLED"); ok {
		ensureHTTPServerConfig(&cfg).Enabled = &value
	}
	if value, ok := envBool("STELLAR_GRPC_ENABLED"); ok {
		ensureGRPCServerConfig(&cfg).Enabled = &value
	}
	return cfg.Normalize()
}

func derefHTTPConfig(value *HTTPConfig) HTTPConfig {
	if value == nil {
		return HTTPConfig{}
	}
	return *value
}

func ensureHTTPServerConfig(cfg *Config) *HTTPServerConfig {
	if cfg.HTTP.Server == nil {
		cfg.HTTP.Server = &HTTPServerConfig{}
	}
	return cfg.HTTP.Server
}

func derefGRPCConfig(value *GRPCConfig) GRPCConfig {
	if value == nil {
		return GRPCConfig{}
	}
	return *value
}

func ensureGRPCServerConfig(cfg *Config) *GRPCServerConfig {
	if cfg.GRPC.Server == nil {
		cfg.GRPC.Server = &GRPCServerConfig{}
	}
	return cfg.GRPC.Server
}

func envBool(key string) (bool, bool) {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return false, false
	}
	parsed, err := strconv.ParseBool(value)
	if err != nil {
		return false, false
	}
	return parsed, true
}

func addrFromPort(addr string, port int, fallback string) string {
	if strings.TrimSpace(addr) != "" {
		return addr
	}
	if port > 0 {
		return ":" + strconv.Itoa(port)
	}
	return fallback
}
