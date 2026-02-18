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
}
