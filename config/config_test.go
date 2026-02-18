package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoad(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	content := `
gateway:
  listen: ":9090"
server:
  listen: ":9091"
backends:
  - path_prefix: /api
    target_url: http://localhost:3000
    name: api
`
	if err := os.WriteFile(path, []byte(content), 0600); err != nil {
		t.Fatal(err)
	}
	cfg, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Gateway.Listen != ":9090" {
		t.Errorf("gateway listen want :9090 got %s", cfg.Gateway.Listen)
	}
	if cfg.Server.Listen != ":9091" {
		t.Errorf("server listen want :9091 got %s", cfg.Server.Listen)
	}
	if len(cfg.Backends) != 1 || cfg.Backends[0].PathPrefix != "/api" {
		t.Errorf("backends want one with /api got %v", cfg.Backends)
	}
}

func TestSetDefaults(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "minimal.yaml")
	if err := os.WriteFile(path, []byte("backends: []"), 0600); err != nil {
		t.Fatal(err)
	}
	cfg, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Gateway.Listen != ":8080" {
		t.Errorf("default gateway listen want :8080 got %s", cfg.Gateway.Listen)
	}
	if cfg.Server.Listen != ":8081" {
		t.Errorf("default server listen want :8081 got %s", cfg.Server.Listen)
	}
}
