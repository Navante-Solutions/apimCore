package gateway

import (
	"crypto/sha256"
	"encoding/hex"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"
	"time"

	"github.com/navantesolutions/apimcore/config"
	"github.com/navantesolutions/apimcore/internal/meter"
	"github.com/navantesolutions/apimcore/internal/store"
)

const (
	HeaderAPIKey    = "X-Api-Key"
	HeaderTenantID  = "X-Tenant-Id"
	HeaderRequestID = "X-Request-Id"
)

type Gateway struct {
	backends []config.BackendRoute
	store    *store.Store
	meter    *meter.Meter
	proxy    *httputil.ReverseProxy
}

func New(cfg *config.Config, s *store.Store, m *meter.Meter) *Gateway {
	return &Gateway{
		backends: cfg.Backends,
		store:    s,
		meter:    m,
		proxy:    &httputil.ReverseProxy{},
	}
}

func (g *Gateway) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	path := r.URL.Path
	var target *config.BackendRoute
	for i := range g.backends {
		if strings.HasPrefix(path, g.backends[i].PathPrefix) {
			target = &g.backends[i]
			break
		}
	}
	if target == nil {
		http.Error(w, "no route for path", http.StatusNotFound)
		g.meter.Record("", "", r.Method, http.StatusNotFound, time.Since(start).Milliseconds(), 0, 0, "")
		return
	}

	apiKey := r.Header.Get(HeaderAPIKey)
	var sub *store.Subscription
	var apiDef *store.ApiDefinition
	if apiKey != "" {
		hash := hashKey(apiKey)
		prefix := apiKey
		if len(prefix) > 8 {
			prefix = prefix[:8]
		}
		k := g.store.GetKeyByHash(hash)
		if k == nil {
			k = g.store.GetKeyByPrefix(prefix)
		}
		if k != nil && k.Active {
			g.store.UpdateKeyLastUsed(k.ID, time.Now())
			sub = g.store.GetSubscription(k.SubscriptionID)
			if sub != nil && sub.Active {
				defs := g.store.ListDefinitionsByProduct(sub.ProductID)
				for i := range defs {
					if strings.HasPrefix(path, defs[i].PathPrefix) {
						apiDef = &defs[i]
						break
					}
				}
			}
		}
	}

	var backendURL string
	var apiDefID int64
	var backendName string
	if apiDef != nil {
		backendURL = apiDef.BackendURL
		apiDefID = apiDef.ID
		backendName = apiDef.Name
	} else {
		backendURL = target.TargetURL
		backendName = target.Name
	}

	targetURL, err := url.Parse(backendURL)
	if err != nil {
		http.Error(w, "bad gateway config", http.StatusInternalServerError)
		g.meter.Record(backendName, target.PathPrefix, r.Method, 502, time.Since(start).Milliseconds(), 0, apiDefID, "")
		return
	}

	rec := &responseRecorder{ResponseWriter: w, status: 200}
	if sub != nil && sub.TenantID != "" {
		r.Header.Set(HeaderTenantID, sub.TenantID)
	}

	dest := *targetURL
	g.proxy.Director = func(req *http.Request) {
		req.URL.Scheme = dest.Scheme
		req.URL.Host = dest.Host
		req.Host = dest.Host
	}
	g.proxy.ServeHTTP(rec, r)

	elapsed := time.Since(start).Milliseconds()
	subID := int64(0)
	tenantID := ""
	if sub != nil {
		subID = sub.ID
		tenantID = sub.TenantID
	}
	g.meter.Record(backendName, target.PathPrefix, r.Method, rec.status, elapsed, subID, apiDefID, tenantID)
	log.Printf("apim gateway: %s %s -> %s %d %dms", r.Method, path, backendName, rec.status, elapsed)
}

func hashKey(key string) string {
	h := sha256.Sum256([]byte(key))
	return hex.EncodeToString(h[:])
}

type responseRecorder struct {
	http.ResponseWriter
	status int
}

func (r *responseRecorder) WriteHeader(code int) {
	r.status = code
	r.ResponseWriter.WriteHeader(code)
}
