package config

import (
	"os"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Gateway   GatewayConfig   `yaml:"gateway"`
	Server    ServerConfig   `yaml:"server"`
	Backends  []BackendRoute `yaml:"backends"`
	DevPortal DevPortalConfig `yaml:"devportal"`
}

type GatewayConfig struct {
	Listen string `yaml:"listen"`
}

type ServerConfig struct {
	Listen string `yaml:"listen"`
}

type BackendRoute struct {
	PathPrefix string `yaml:"path_prefix"`
	TargetURL  string `yaml:"target_url"`
	Name      string `yaml:"name"`
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
