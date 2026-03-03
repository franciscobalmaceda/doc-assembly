package config

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"strings"

	"github.com/spf13/viper"
)

// Load reads configuration from YAML files and environment variables.
// Environment variables take precedence over YAML values.
// Env prefix: DOC_ENGINE_ (e.g., DOC_ENGINE_SERVER_PORT)
func Load() (*Config, error) {
	v := viper.New()

	// Set config file settings
	v.SetConfigName("app")
	v.SetConfigType("yaml")

	// Add config paths (searched in order)
	v.AddConfigPath("./settings")
	v.AddConfigPath("../settings")
	v.AddConfigPath("../../settings")
	v.AddConfigPath(".")

	// Environment variable settings
	v.SetEnvPrefix("DOC_ENGINE")
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	v.AutomaticEnv()

	// Read config file
	if err := v.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return nil, fmt.Errorf("reading config file: %w", err)
		}
		// Config file not found is acceptable, we'll use env vars and defaults
	}

	// Set defaults
	setDefaults(v)

	// Unmarshal into struct
	var cfg Config
	if err := v.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("unmarshaling config: %w", err)
	}

	// Special handling for PORT env var (common in container environments)
	if cfg.Server.Port == "" {
		if port := os.Getenv("PORT"); port != "" {
			cfg.Server.Port = port
		}
	}

	// Explicit env override for dummy auth flag.
	// Viper's AutomaticEnv does not reliably propagate nested env vars during Unmarshal.
	if os.Getenv("DOC_ENGINE_AUTH_DUMMY") == "true" {
		cfg.Auth.Dummy = true
	}

	// Set default DummyAuthUserID when in dummy auth mode.
	// Uses the well-known dummy UUID; override with DOC_ENGINE_DUMMY_AUTH_USER_ID env var.
	if cfg.Auth.IsDummyAuth() && cfg.DummyAuthUserID == "" {
		if id := os.Getenv("DOC_ENGINE_DUMMY_AUTH_USER_ID"); id != "" {
			cfg.DummyAuthUserID = id
		} else {
			cfg.DummyAuthUserID = "00000000-0000-0000-0000-000000000001"
		}
	}

	// Explicit env overrides for nested config (same Viper nested env issue).
	applySigningEnvOverrides(&cfg.Signing)
	applyAuthPanelEnvOverrides(cfg.Auth.Panel)
	applyBootstrapEnvOverrides(&cfg.Bootstrap)

	// Run OIDC discovery to populate issuer/jwks_url from discovery endpoints.
	// Non-fatal: dev mode (no OIDC) and manual config still work.
	if err := cfg.Auth.DiscoverAll(context.Background()); err != nil {
		slog.WarnContext(context.Background(), "OIDC discovery failed (non-fatal)",
			slog.String("error", err.Error()))
	}

	return &cfg, nil
}

// setDefaults sets default configuration values.
func setDefaults(v *viper.Viper) {
	// Server defaults
	v.SetDefault("server.port", "8080")
	v.SetDefault("server.base_path", "")
	v.SetDefault("server.read_timeout", 30)
	v.SetDefault("server.write_timeout", 30)
	v.SetDefault("server.shutdown_timeout", 10)

	// Database defaults
	v.SetDefault("database.host", "localhost")
	v.SetDefault("database.port", 5432)
	v.SetDefault("database.user", "postgres")
	v.SetDefault("database.name", "doc_engine")
	v.SetDefault("database.ssl_mode", "disable")
	v.SetDefault("database.max_pool_size", 10)
	v.SetDefault("database.min_pool_size", 2)
	v.SetDefault("database.max_idle_time", 300)

	// Logging defaults
	v.SetDefault("logging.level", "info")
	v.SetDefault("logging.format", "json")

	// Bootstrap defaults
	v.SetDefault("bootstrap.enabled", true)

	// Internal API defaults
	v.SetDefault("internal_api.enabled", true)

	// Environment default
	v.SetDefault("environment", "development")

	// Environment aliases — map environment names to canonical dev/prod
	v.SetDefault("environment_aliases", map[string][]string{
		"dev":  {"dev", "develop", "development", "staging", "uat", "qa", "local", "sandbox"},
		"prod": {"prod", "production"},
	})
}

// applySigningEnvOverrides reads DOC_ENGINE_SIGNING_* env vars into SigningConfig.
func applySigningEnvOverrides(cfg *SigningConfig) {
	if v := os.Getenv("DOC_ENGINE_SIGNING_PROVIDER"); v != "" {
		cfg.Provider = v
	}
	if v := os.Getenv("DOC_ENGINE_SIGNING_API_KEY"); v != "" {
		cfg.APIKey = v
	}
	if v := os.Getenv("DOC_ENGINE_SIGNING_BASE_URL"); v != "" {
		cfg.BaseURL = v
	}
	if v := os.Getenv("DOC_ENGINE_SIGNING_SIGNING_BASE_URL"); v != "" {
		cfg.SigningBaseURL = v
	}
	if v := os.Getenv("DOC_ENGINE_SIGNING_WEBHOOK_SECRET"); v != "" {
		cfg.WebhookSecret = v
	}
	if v := os.Getenv("DOC_ENGINE_SIGNING_WEBHOOK_URL"); v != "" {
		cfg.WebhookURL = v
	}
}

// applyAuthPanelEnvOverrides reads DOC_ENGINE_AUTH_PANEL_* env vars into OIDCProvider.
func applyAuthPanelEnvOverrides(panel *OIDCProvider) {
	if panel == nil {
		return
	}
	if v := os.Getenv("DOC_ENGINE_AUTH_PANEL_NAME"); v != "" {
		panel.Name = v
	}
	if v := os.Getenv("DOC_ENGINE_AUTH_PANEL_DISCOVERY_URL"); v != "" {
		panel.DiscoveryURL = v
	}
	if v := os.Getenv("DOC_ENGINE_AUTH_PANEL_ISSUER"); v != "" {
		panel.Issuer = v
	}
	if v := os.Getenv("DOC_ENGINE_AUTH_PANEL_AUDIENCE"); v != "" {
		panel.Audience = v
	}
	if v := os.Getenv("DOC_ENGINE_AUTH_PANEL_CLIENT_ID"); v != "" {
		panel.ClientID = v
	}
}

// applyBootstrapEnvOverrides reads DOC_ENGINE_BOOTSTRAP_* env vars into BootstrapConfig.
func applyBootstrapEnvOverrides(cfg *BootstrapConfig) {
	if v := os.Getenv("DOC_ENGINE_BOOTSTRAP_ENABLED"); v == "false" {
		cfg.Enabled = false
	}
}

// LoadFromFile reads configuration from a specific YAML file path.
// Environment variables still apply as overrides.
func LoadFromFile(filePath string) (*Config, error) {
	v := viper.New()

	v.SetConfigFile(filePath)

	// Environment variable settings
	v.SetEnvPrefix("DOC_ENGINE")
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	v.AutomaticEnv()

	if err := v.ReadInConfig(); err != nil {
		return nil, fmt.Errorf("reading config file %s: %w", filePath, err)
	}

	setDefaults(v)

	var cfg Config
	if err := v.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("unmarshaling config: %w", err)
	}

	if cfg.Server.Port == "" {
		if port := os.Getenv("PORT"); port != "" {
			cfg.Server.Port = port
		}
	}

	if os.Getenv("DOC_ENGINE_AUTH_DUMMY") == "true" {
		cfg.Auth.Dummy = true
	}

	if cfg.Auth.IsDummyAuth() && cfg.DummyAuthUserID == "" {
		if id := os.Getenv("DOC_ENGINE_DUMMY_AUTH_USER_ID"); id != "" {
			cfg.DummyAuthUserID = id
		} else {
			cfg.DummyAuthUserID = "00000000-0000-0000-0000-000000000001"
		}
	}

	applySigningEnvOverrides(&cfg.Signing)
	applyAuthPanelEnvOverrides(cfg.Auth.Panel)
	applyBootstrapEnvOverrides(&cfg.Bootstrap)

	if err := cfg.Auth.DiscoverAll(context.Background()); err != nil {
		slog.WarnContext(context.Background(), "OIDC discovery failed (non-fatal)",
			slog.String("error", err.Error()))
	}

	return &cfg, nil
}

// MustLoad loads configuration and panics on error.
// Use this only in main() or initialization code.
func MustLoad() *Config {
	cfg, err := Load()
	if err != nil {
		panic(fmt.Sprintf("failed to load config: %v", err))
	}
	return cfg
}
