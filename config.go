package stellflux

const (
	EnvDev  Environment = "dev"
	EnvUAT  Environment = "uat"
	EnvPre  Environment = "pre"
	EnvProd Environment = "prod"
)

type Environment string

type Config struct {
	AppName     string
	Environment Environment
	Zone        string
	Disabled    bool
}

func (c Config) Normalize() Config {
	if c.Environment == "" {
		c.Environment = EnvDev
	}
	return c
}
