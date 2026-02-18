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
products:
  - name: "Test Product"
    slug: "test-slug"
    description: "description"
    apis:
      - name: "testapi"
        path_prefix: "/api"
        target_url: "http://localhost:3000"
subscriptions:
  - developer_id: "dev1"
    product_slug: "test-slug"
    plan: "free"
    keys:
      - name: "key1"
        value: "val1"
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
	if len(cfg.Products) != 1 || cfg.Products[0].Slug != "test-slug" {
		t.Errorf("products want test-slug got %v", cfg.Products)
	}
	if len(cfg.Products[0].Apis) != 1 || cfg.Products[0].Apis[0].PathPrefix != "/api" {
		t.Errorf("apis want /api got %v", cfg.Products[0].Apis)
	}
	if len(cfg.Subscriptions) != 1 || cfg.Subscriptions[0].ProductSlug != "test-slug" {
		t.Errorf("subscriptions want test-slug got %v", cfg.Subscriptions)
	}
}

func TestSetDefaults(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "minimal.yaml")
	if err := os.WriteFile(path, []byte("products: []"), 0600); err != nil {
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
