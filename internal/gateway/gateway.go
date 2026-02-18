package gateway

import (
	"crypto/sha256"
	"encoding/hex"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"
	"sync"
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

type TrafficPacket struct {
	Timestamp time.Time
	Method    string
	Path      string
	Backend   string
	Status    int
	Latency   int64
	TenantID  string
}

type Gateway struct {
	mu          sync.RWMutex
	config      *config.Config
	store       *store.Store
	meter       *meter.Meter
	proxy       *httputil.ReverseProxy
	TrafficChan chan TrafficPacket
}

func New(cfg *config.Config, s *store.Store, m *meter.Meter) *Gateway {
	return &Gateway{
		config:      cfg,
		store:       s,
		meter:       m,
		proxy:       &httputil.ReverseProxy{},
		TrafficChan: make(chan TrafficPacket, 100),
	}
}

func (g *Gateway) UpdateConfig(cfg *config.Config) {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.config = cfg
}

func (g *Gateway) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	g.mu.RLock()
	defer g.mu.RUnlock()

	start := time.Now()
	path := r.URL.Path

	var targetApi *config.ApiConfig

	for i := range g.config.Products {
		p := &g.config.Products[i]
		for j := range p.Apis {
			a := &p.Apis[j]
			if strings.HasPrefix(path, a.PathPrefix) {
				targetApi = a
				break
			}
		}
		if targetApi != nil {
			break
		}
	}

	if targetApi == nil {
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
		backendURL = targetApi.BackendURL
		backendName = targetApi.Name
	}

	targetURL, err := url.Parse(backendURL)
	if err != nil {
		http.Error(w, "bad gateway config", http.StatusInternalServerError)
		g.meter.Record(backendName, targetApi.PathPrefix, r.Method, 502, time.Since(start).Milliseconds(), 0, apiDefID, "")
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
	g.meter.Record(backendName, targetApi.PathPrefix, r.Method, rec.status, elapsed, subID, apiDefID, tenantID)

	// Emit traffic packet for TUI
	select {
	case g.TrafficChan <- TrafficPacket{
		Timestamp: start,
		Method:    r.Method,
		Path:      path,
		Backend:   backendName,
		Status:    rec.status,
		Latency:   elapsed,
		TenantID:  tenantID,
	}:
	default:
		// Drop if channel full
	}

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
