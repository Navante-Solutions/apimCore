package config

import (
	"os"

	"gopkg.in/yaml.v3"
)

const (
	DefaultGatewayListen        = ":8080"
	DefaultServerListen         = ":8081"
	DefaultBackendTimeoutSec    = 30
	DefaultDevPortalPath        = "/devportal"
)

type Config struct {
	Gateway       GatewayConfig        `yaml:"gateway"`
	Server        ServerConfig         `yaml:"server"`
	Products      []ProductConfig      `yaml:"products"`
	Subscriptions []SubscriptionConfig `yaml:"subscriptions"`
	DevPortal     DevPortalConfig      `yaml:"devportal"`
	Security      SecurityConfig       `yaml:"security"`
}

type GatewayConfig struct {
	Listen               string `yaml:"listen"`
	BackendTimeoutSeconds int    `yaml:"backend_timeout_seconds"`
}

type ServerConfig struct {
	Listen string `yaml:"listen"`
}

type ProductConfig struct {
	Name        string      `yaml:"name"`
	Slug        string      `yaml:"slug"`
	Description string      `yaml:"description"`
	Apis        []ApiConfig `yaml:"apis"`
}

type ApiConfig struct {
	Name            string            `yaml:"name"`
	Host            string            `yaml:"host"`
	PathPrefix      string            `yaml:"path_prefix"`
	BackendURL      string            `yaml:"target_url"`
	OpenAPISpecURL  string            `yaml:"openapi_spec_url"`
	Version         string            `yaml:"version"`
	AddHeaders      map[string]string  `yaml:"add_headers"`
	StripPathPrefix bool              `yaml:"strip_path_prefix"`
}

type SubscriptionConfig struct {
	DeveloperID string      `yaml:"developer_id"`
	ProductID   int64       `yaml:"product_id"`
	ProductSlug string      `yaml:"product_slug"`
	TenantID    string      `yaml:"tenant_id"`
	Plan        string      `yaml:"plan"`
	Keys        []KeyConfig `yaml:"keys"`
}

type KeyConfig struct {
	Name  string `yaml:"name"`
	Value string `yaml:"value"`
}

type DevPortalConfig struct {
	Enabled bool   `yaml:"enabled"`
	Path    string `yaml:"path"`
}

type SecurityConfig struct {
	IPBlacklist      []string        `yaml:"ip_blacklist"`
	AllowedCountries []string        `yaml:"allowed_countries"`
	RateLimit        RateLimitConfig `yaml:"rate_limit"`
}

type RateLimitConfig struct {
	Enabled bool    `yaml:"enabled"`
	RPS     float64 `yaml:"requests_per_second"`
	Burst   int     `yaml:"burst"`
}

func Default() *Config {
	c := &Config{}
	setDefaults(c)
	return c
}

func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var c Config
	if err := yaml.Unmarshal(data, &c); err != nil {
		return nil, err
	}
	setDefaults(&c)
	return &c, nil
}

func setDefaults(c *Config) {
	if c.Gateway.Listen == "" {
		c.Gateway.Listen = DefaultGatewayListen
	}
	if c.Gateway.BackendTimeoutSeconds <= 0 {
		c.Gateway.BackendTimeoutSeconds = DefaultBackendTimeoutSec
	}
	if c.Server.Listen == "" {
		c.Server.Listen = DefaultServerListen
	}
	if c.DevPortal.Path == "" {
		c.DevPortal.Path = DefaultDevPortalPath
	}
	if c.DevPortal.Enabled {
		c.DevPortal.Enabled = true
	}
	if v := os.Getenv("APIM_GATEWAY_LISTEN"); v != "" {
		c.Gateway.Listen = v
	}
	if v := os.Getenv("APIM_SERVER_LISTEN"); v != "" {
		c.Server.Listen = v
	}
}
