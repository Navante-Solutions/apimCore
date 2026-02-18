package config

import (
	"os"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Gateway       GatewayConfig        `yaml:"gateway"`
	Server        ServerConfig         `yaml:"server"`
	Products      []ProductConfig      `yaml:"products"`
	Subscriptions []SubscriptionConfig `yaml:"subscriptions"`
	DevPortal     DevPortalConfig      `yaml:"devportal"`
}

type GatewayConfig struct {
	Listen string `yaml:"listen"`
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
	Name           string `yaml:"name"`
	PathPrefix     string `yaml:"path_prefix"`
	BackendURL     string `yaml:"target_url"`
	OpenAPISpecURL string `yaml:"openapi_spec_url"`
	Version        string `yaml:"version"`
}

type SubscriptionConfig struct {
	DeveloperID string      `yaml:"developer_id"`
	ProductID   int64       `yaml:"product_id"`   // Internal ID mapping
	ProductSlug string      `yaml:"product_slug"` // YAML lookup
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
		c.Gateway.Listen = ":8080"
	}
	if c.Server.Listen == "" {
		c.Server.Listen = ":8081"
	}
	if c.DevPortal.Path == "" {
		c.DevPortal.Path = "/devportal"
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
