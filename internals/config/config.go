package config

import (
	"fmt"
	"os"
	"strings"

	"github.com/knadh/koanf/parsers/yaml"
	"github.com/knadh/koanf/providers/env/v2"
	"github.com/knadh/koanf/providers/file"
	"github.com/knadh/koanf/v2"
)

const (
	EnvPrefix         = "2CHI_"
	DefaultConfigPath = "./config.yaml"
	MinAuthSecretLen  = 32
)

type Config struct {
	App      AppConfig      `koanf:"app"`
	Server   ServerConfig   `koanf:"server"`
	Logger   LoggerConfig   `koanf:"logger"`
	Auth     AuthConfig     `koanf:"auth"`
	Postgres PostgresConfig `koanf:"postgres"`
	Redis    RedisConfig    `koanf:"redis"`

	AWS     AWSConfig     `koanf:"aws"`
	Jobs    JobsConfig    `koanf:"jobs"`
	Support SupportConfig `koanf:"support"`
	Google  GoogleConfig  `koanf:"google"`
	Paddle  PaddleConfig  `koanf:"paddle"`
}

type AppConfig struct {
	Name           string `koanf:"name"`
	Namespace      string `koanf:"namespace"`
	WebURL         string `koanf:"web_url"`
	CookieDomain   string `koanf:"cookie_domain"`
	Environment    string `koanf:"environment"`
	MigrationsPath string `koanf:"migrations_path"`
	LocalesPath    string `koanf:"locales_path"`
	TemplatesPath  string `koanf:"templates_path"`
}

type ServerConfig struct {
	Port               int      `koanf:"port"`
	MetricsPort        int      `koanf:"metrics_port"`
	CORSAllowOrigins   []string `koanf:"cors_allow_origins"`
	BodyLimit          string   `koanf:"body_limit"`
	ServerReadTimeout  int      `koanf:"server_read_timeout"`
	ServerWriteTimeout int      `koanf:"server_write_timeout"`
	ServerIdleTimeout  int      `koanf:"server_idle_timeout"`
}

type LoggerConfig struct {
	Level  string `koanf:"level"`
	Format string `koanf:"format"`
}

type AuthConfig struct {
	AccessTokenSecret string `koanf:"access_token_secret"`
	TokenHashPepper   string `koanf:"token_hash_pepper"`
}

type PostgresConfig struct {
	DSN             string `koanf:"dsn"`
	MaxOpenConns    int    `koanf:"max_open_conns"`
	MaxIdleConns    int    `koanf:"max_idle_conns"`
	ConnMaxLifetime int    `koanf:"conn_max_lifetime"`
	ConnMaxIdleTime int    `koanf:"conn_max_idle_time"`
	PingTimeout     int    `koanf:"ping_timeout"`
}

type RedisConfig struct {
	SessionDSN   string `koanf:"session_dsn"`
	RateLimitDSN string `koanf:"rate_limit_dsn"`
}

type AWSConfig struct {
	DefaultRegion   string    `koanf:"default_region"`
	DefaultEndpoint string    `koanf:"default_endpoint"`
	Ses             SesConfig `koanf:"ses"`
}

type SesConfig struct {
	FromEmail string `koanf:"from_email"`
	FromName  string `koanf:"from_name"`
}

type JobsConfig struct {
	MaxRetries                int            `koanf:"max_retries"`
	ApplyScheduledPlanChanges JobQueueConfig `koanf:"apply_scheduled_plan_changes"`
}

type JobQueueConfig struct {
	Concurrency int    `koanf:"concurrency"`
	QueueURL    string `koanf:"queue_url"`
}

type SupportConfig struct {
	InboxEmail string `koanf:"inbox_email"`
}

type GoogleConfig struct {
	OAuth OAuthConfig `koanf:"oauth"`
	Maps  MapsConfig  `koanf:"maps"`
}

type OAuthConfig struct {
	RedirectURL  string `koanf:"redirect_url"`
	ClientID     string `koanf:"client_id"`
	ClientSecret string `koanf:"client_secret"`
}

type MapsConfig struct {
	APIKey string `koanf:"api_key"`
}

type PaddleConfig struct {
	Environment   string             `koanf:"environment"`
	APIKey        string             `koanf:"api_key"`
	WebhookSecret string             `koanf:"webhook_secret"`
	Prices        PaddlePricesConfig `koanf:"prices"`
}

type PaddlePricesConfig struct {
	BasicMonthly string `koanf:"basic_monthly"`
	BasicAnnual  string `koanf:"basic_annual"`
	ProMonthly   string `koanf:"pro_monthly"`
	ProAnnual    string `koanf:"pro_annual"`
}

// Init loads non-secret config from YAML and secret values from process
// environment variables (e.g. injected by Docker, Kubernetes, or the shell).
func Init() (*Config, error) {
	configPath := os.Getenv("CONFIG_PATH")
	if configPath == "" {
		configPath = DefaultConfigPath
	}

	k := koanf.New(".")

	if err := k.Load(file.Provider(configPath), yaml.Parser()); err != nil {
		return nil, fmt.Errorf("config: load yaml file at path %s: %w", configPath, err)
	}

	if err := k.Load(env.Provider(".", env.Opt{
		Prefix:        EnvPrefix,
		TransformFunc: envToKoanfKey,
	}), nil); err != nil {
		return nil, fmt.Errorf("config: load environment: %w", err)
	}

	var cfg Config
	if err := k.UnmarshalWithConf("", &cfg, koanf.UnmarshalConf{Tag: "koanf"}); err != nil {
		return nil, fmt.Errorf("config: unmarshal: %w", err)
	}

	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	return &cfg, nil
}

// Validate checks required configuration before the process starts serving traffic.
func (c *Config) Validate() error {
	if len(c.Auth.AccessTokenSecret) < MinAuthSecretLen {
		return fmt.Errorf("config: auth.access_token_secret must be at least %d characters", MinAuthSecretLen)
	}
	if len(c.Auth.TokenHashPepper) < MinAuthSecretLen {
		return fmt.Errorf("config: auth.token_hash_pepper must be at least %d characters", MinAuthSecretLen)
	}
	if c.Paddle.APIKey != "" {
		if err := c.Paddle.Prices.validate(); err != nil {
			return err
		}
	}
	return nil
}

func (p PaddlePricesConfig) validate() error {
	missing := make([]string, 0, 4)
	if p.BasicMonthly == "" {
		missing = append(missing, "basic_monthly")
	}
	if p.BasicAnnual == "" {
		missing = append(missing, "basic_annual")
	}
	if p.ProMonthly == "" {
		missing = append(missing, "pro_monthly")
	}
	if p.ProAnnual == "" {
		missing = append(missing, "pro_annual")
	}
	if len(missing) > 0 {
		return fmt.Errorf(
			"config: paddle.prices (%s) required when paddle.api_key is set (must match react-spa VITE_PADDLE_PRICE_*)",
			strings.Join(missing, ", "),
		)
	}
	return nil
}

// secretEnvSections limits which YAML sections can be overridden by env vars.
// Only put secrets in env; everything else stays in config.yaml.
var secretEnvSections = map[string]struct{}{
	"postgres": {},
	"redis":    {},
	"auth":     {},
	"google":   {},
	"paddle":   {},
}

// envToKoanfKey maps env vars to koanf paths using a naming convention:
// use __ (double underscore) for nesting, _ (single) for snake_case field names.
func envToKoanfKey(envKey, envValue string) (string, any) {
	key := strings.TrimPrefix(envKey, EnvPrefix)
	key = strings.ToLower(key)
	key = strings.ReplaceAll(key, "__", ".")

	topLevel, _, ok := strings.Cut(key, ".")
	if !ok {
		return "", nil
	}
	if _, allowed := secretEnvSections[topLevel]; !allowed {
		return "", nil
	}

	return key, strings.TrimSpace(envValue)
}
