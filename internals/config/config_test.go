package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestInit_SecretEnvOverridesOnly(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, "config.yaml")

	yaml := []byte(`server:
  port: 1300
logger:
  level: "info"
  format: "json"

postgres:
  max_open_conns: 1
  max_idle_conns: 1
  conn_max_lifetime: 1
  conn_max_idle_time: 1
  ping_timeout: 1

redis:
  session_dsn: ""
  rate_limit_dsn: ""

google:
  oauth:
    redirect_url: "http://localhost/auth/google/callback"

paddle:
  environment: "sandbox"
`)
	if err := os.WriteFile(configPath, yaml, 0o600); err != nil {
		t.Fatalf("write yaml: %v", err)
	}

	// Make Init read our temp config.
	t.Setenv("CONFIG_PATH", configPath)

	// Non-secret env var should be ignored so YAML remains the source of truth.
	t.Setenv(EnvPrefix+"SERVER__PORT", "9999")

	// Secret env vars should override YAML (use __ for nesting).
	t.Setenv(EnvPrefix+"POSTGRES__DSN", "postgres://secret")
	t.Setenv(EnvPrefix+"REDIS__SESSION_DSN", "redis://session")
	t.Setenv(EnvPrefix+"REDIS__RATE_LIMIT_DSN", "redis://rate")

	t.Setenv(EnvPrefix+"AUTH__ACCESS_TOKEN_SECRET", "access-secret-with-at-least-32-characters")
	t.Setenv(EnvPrefix+"AUTH__TOKEN_HASH_PEPPER", "pepper-with-at-least-32-characters")

	t.Setenv(EnvPrefix+"GOOGLE__OAUTH__CLIENT_ID", "client-id")
	t.Setenv(EnvPrefix+"GOOGLE__OAUTH__CLIENT_SECRET", "client-secret")

	t.Setenv(EnvPrefix+"PADDLE__API_KEY", "paddle-api-key")
	t.Setenv(EnvPrefix+"PADDLE__WEBHOOK_SECRET", "paddle-webhook-secret")
	t.Setenv(EnvPrefix+"GOOGLE__MAPS__API_KEY", "google-maps-api-key")

	cfg, err := Init()
	if err != nil {
		t.Fatalf("Init() error = %v", err)
	}

	if got, want := cfg.Server.Port, 1300; got != want {
		t.Fatalf("cfg.Server.Port = %d, want %d", got, want)
	}

	if got, want := cfg.Postgres.DSN, "postgres://secret"; got != want {
		t.Fatalf("cfg.Postgres.DSN = %q, want %q", got, want)
	}
	if got, want := cfg.Redis.SessionDSN, "redis://session"; got != want {
		t.Fatalf("cfg.Redis.SessionDSN = %q, want %q", got, want)
	}
	if got, want := cfg.Redis.RateLimitDSN, "redis://rate"; got != want {
		t.Fatalf("cfg.Redis.RateLimitDSN = %q, want %q", got, want)
	}

	if got, want := cfg.Auth.AccessTokenSecret, "access-secret-with-at-least-32-characters"; got != want {
		t.Fatalf("cfg.Auth.AccessTokenSecret = %q, want %q", got, want)
	}
	if got, want := cfg.Auth.TokenHashPepper, "pepper-with-at-least-32-characters"; got != want {
		t.Fatalf("cfg.Auth.TokenHashPepper = %q, want %q", got, want)
	}

	if got, want := cfg.Google.OAuth.RedirectURL, "http://localhost/auth/google/callback"; got != want {
		t.Fatalf("cfg.Google.OAuth.RedirectURL = %q, want %q", got, want)
	}
	if got, want := cfg.Google.OAuth.ClientID, "client-id"; got != want {
		t.Fatalf("cfg.Google.OAuth.ClientID = %q, want %q", got, want)
	}
	if got, want := cfg.Google.OAuth.ClientSecret, "client-secret"; got != want {
		t.Fatalf("cfg.Google.OAuth.ClientSecret = %q, want %q", got, want)
	}
	if got, want := cfg.Google.Maps.APIKey, "google-maps-api-key"; got != want {
		t.Fatalf("cfg.Google.Maps.APIKey = %q, want %q", got, want)
	}

	if got, want := cfg.Paddle.Environment, "sandbox"; got != want {
		t.Fatalf("cfg.Paddle.Environment = %q, want %q", got, want)
	}
	if got, want := cfg.Paddle.APIKey, "paddle-api-key"; got != want {
		t.Fatalf("cfg.Paddle.APIKey = %q, want %q", got, want)
	}
	if got, want := cfg.Paddle.WebhookSecret, "paddle-webhook-secret"; got != want {
		t.Fatalf("cfg.Paddle.WebhookSecret = %q, want %q", got, want)
	}
}

func TestValidate_AuthSecretsTooShort(t *testing.T) {
	tests := []struct {
		name    string
		secret  string
		pepper  string
		wantErr string
	}{
		{
			name:    "empty access token secret",
			secret:  "",
			pepper:  "pepper-with-at-least-32-characters-long",
			wantErr: "auth.access_token_secret",
		},
		{
			name:    "short token hash pepper",
			secret:  "access-secret-with-at-least-32-characters",
			pepper:  "short",
			wantErr: "auth.token_hash_pepper",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &Config{
				Auth: AuthConfig{
					AccessTokenSecret: tt.secret,
					TokenHashPepper:   tt.pepper,
				},
			}
			err := cfg.Validate()
			if err == nil {
				t.Fatal("Validate() expected error")
			}
			if !strings.Contains(err.Error(), tt.wantErr) {
				t.Fatalf("Validate() error = %q, want substring %q", err.Error(), tt.wantErr)
			}
		})
	}
}
