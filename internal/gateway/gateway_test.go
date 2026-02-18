package gateway

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/prometheus/client_golang/prometheus"

	"github.com/navantesolutions/apimcore/config"
	"github.com/navantesolutions/apimcore/internal/meter"
	"github.com/navantesolutions/apimcore/internal/store"
)

func TestGateway_ServeHTTP(t *testing.T) {
	// 1. Setup mock backend
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Test-Backend", "ok")
		if tid := r.Header.Get(HeaderTenantID); tid != "" {
			w.Header().Set("X-Detected-Tenant", tid)
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer backend.Close()

	// 2. Setup store and config
	s := store.NewStore()
	cfg := &config.Config{
		Products: []config.ProductConfig{
			{
				Slug: "p1",
				Apis: []config.ApiConfig{
					{
						Name:       "api1",
						PathPrefix: "/api1",
						BackendURL: backend.URL,
					},
				},
			},
		},
		Subscriptions: []config.SubscriptionConfig{
			{
				DeveloperID: "dev1",
				ProductSlug: "p1",
				Keys: []config.KeyConfig{
					{Name: "valid-key", Value: "secret123"},
				},
			},
		},
	}
	s.PopulateFromConfig(cfg)

	// 3. Setup Gateway
	reg := prometheus.NewRegistry()
	m := meter.New(s, reg)
	gw := New(cfg, s, m)

	t.Run("Valid Key and Routing", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api1/hello", nil)
		req.Header.Set(HeaderAPIKey, "secret123")
		rec := httptest.NewRecorder()

		gw.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("expected 200, got %d", rec.Code)
		}
		if rec.Header().Get("X-Test-Backend") != "ok" {
			t.Error("request did not reach backend")
		}
	})

	t.Run("Invalid Routing", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/unknown", nil)
		rec := httptest.NewRecorder()

		gw.ServeHTTP(rec, req)

		if rec.Code != http.StatusNotFound {
			t.Errorf("expected 404, got %d", rec.Code)
		}
	})

	t.Run("No API Key (Fallback to Target Config)", func(t *testing.T) {
		// Even without a key, it should route if the target is defined in products
		// but won't have tenant context.
		req := httptest.NewRequest("GET", "/api1/hello", nil)
		rec := httptest.NewRecorder()

		gw.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("expected 200, got %d", rec.Code)
		}
		if rec.Header().Get("X-Detected-Tenant") != "" {
			t.Error("did not expect tenant ID without key")
		}
	})

	t.Run("IP Blacklist", func(t *testing.T) {
		cfg.Security.IPBlacklist = []string{"1.2.3.4", "192.168.1.0/24"}
		gw.UpdateConfig(cfg)

		tests := []struct {
			name   string
			remote string
			status int
		}{
			{"Blocked IP", "1.2.3.4:1234", http.StatusForbidden},
			{"Blocked CIDR", "192.168.1.10:5555", http.StatusForbidden},
			{"Allowed IP", "5.5.5.5:80", http.StatusOK},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				req := httptest.NewRequest("GET", "/api1/hello", nil)
				req.RemoteAddr = tt.remote
				rec := httptest.NewRecorder()
				gw.ServeHTTP(rec, req)
				if rec.Code != tt.status {
					t.Errorf("%s: expected %d, got %d", tt.name, tt.status, rec.Code)
				}
			})
		}
		// Reset
		cfg.Security.IPBlacklist = nil
		gw.UpdateConfig(cfg)
	})

	t.Run("Rate Limiting", func(t *testing.T) {
		cfg.Security.RateLimit = config.RateLimitConfig{
			Enabled: true,
			RPP:     100, // Higher limit for testing other things if needed
			Burst:   100,
		}
		gw.UpdateConfig(cfg)

		req := httptest.NewRequest("GET", "/api1/hello", nil)
		req.RemoteAddr = "10.0.0.1:1234"

		// 1st request: OK
		rec1 := httptest.NewRecorder()
		gw.ServeHTTP(rec1, req)
		if rec1.Code != http.StatusOK {
			t.Errorf("1st request: expected 200, got %d", rec1.Code)
		}

		// Now force a 429 with tight limits
		cfg.Security.RateLimit = config.RateLimitConfig{
			Enabled: true,
			RPP:     0.1,
			Burst:   1,
		}
		gw.UpdateConfig(cfg)

		// 1st request (consumes the burst token)
		rec2 := httptest.NewRecorder()
		gw.ServeHTTP(rec2, req)
		if rec2.Code != http.StatusOK {
			t.Errorf("2nd request (1st with limit): expected 200, got %d", rec2.Code)
		}

		// 2nd request (immediate, no tokens left)
		rec3 := httptest.NewRecorder()
		gw.ServeHTTP(rec3, req)
		if rec3.Code != http.StatusTooManyRequests {
			t.Errorf("3rd request: expected 429, got %d", rec3.Code)
		}

		// Reset for next tests
		cfg.Security.RateLimit.Enabled = false
		gw.UpdateConfig(cfg)
	})

	t.Run("GeoIP Logic", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api1/hello", nil)
		req.RemoteAddr = "8.8.8.8:1234"
		rec := httptest.NewRecorder()
		gw.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("expected 200, got %d", rec.Code)
		}
	})

	t.Run("Host-based Routing", func(t *testing.T) {
		cfg.Products = append(cfg.Products, config.ProductConfig{
			Slug: "p2",
			Apis: []config.ApiConfig{
				{
					Name:       "api-host",
					Host:       "api.example.com",
					PathPrefix: "/",
					BackendURL: backend.URL,
				},
				{
					Name:       "api-wildcard",
					Host:       "*.dev.local",
					PathPrefix: "/",
					BackendURL: backend.URL,
				},
			},
		})
		s.PopulateFromConfig(cfg)

		tests := []struct {
			name   string
			host   string
			path   string
			status int
		}{
			{"Exact Host Match", "api.example.com", "/anything", http.StatusOK},
			{"Wildcard Subdomain Match", "service1.dev.local", "/hi", http.StatusOK},
			{"Wildcard Nested Match", "foo.bar.dev.local", "/deep", http.StatusOK},
			{"Host Mismatch", "other.com", "/api-host", http.StatusNotFound},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				req := httptest.NewRequest("GET", tt.path, nil)
				req.Host = tt.host
				rec := httptest.NewRecorder()
				gw.ServeHTTP(rec, req)
				if rec.Code != tt.status {
					t.Errorf("host %s: expected %d, got %d", tt.host, tt.status, rec.Code)
				}
			})
		}
	})
}
