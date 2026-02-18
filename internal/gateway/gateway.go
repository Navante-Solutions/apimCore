package gateway

import (
	"crypto/sha256"
	"encoding/hex"
	"log"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/navantesolutions/apimcore/config"
	"github.com/navantesolutions/apimcore/internal/hub"
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
	Country   string
}

type Gateway struct {
	mu      sync.RWMutex
	config  *config.Config
	store   *store.Store
	meter   *meter.Meter
	proxy   *httputil.ReverseProxy
	handler http.Handler
	Hub     *hub.Broadcaster
	// Security controls
	securityMu sync.Mutex
	blacklist  map[string]bool
	blacknets  []*net.IPNet
	allowedGeo map[string]bool
}

func New(cfg *config.Config, s *store.Store, m *meter.Meter, h *hub.Broadcaster) *Gateway {
	g := &Gateway{
		config: cfg,
		store:  s,
		meter:  m,
		proxy:  &httputil.ReverseProxy{},
		Hub:    h,
	}
	g.UpdateSecurity(cfg.Security)
	g.rebuildHandler()
	return g
}

func (g *Gateway) UpdateSecurity(cfg config.SecurityConfig) {
	g.securityMu.Lock()
	defer g.securityMu.Unlock()

	g.blacklist = make(map[string]bool)
	g.blacknets = make([]*net.IPNet, 0)
	g.allowedGeo = make(map[string]bool)

	for _, entry := range cfg.IPBlacklist {
		if _, ipnet, err := net.ParseCIDR(entry); err == nil {
			g.blacknets = append(g.blacknets, ipnet)
		} else if ip := net.ParseIP(entry); ip != nil {
			g.blacklist[ip.String()] = true
		}
	}

	for _, country := range cfg.AllowedCountries {
		g.allowedGeo[country] = true
	}
}

func (g *Gateway) GetSecurity() config.SecurityConfig {
	g.securityMu.Lock()
	defer g.securityMu.Unlock()

	blacklist := make([]string, 0, len(g.blacklist)+len(g.blacknets))
	for ip := range g.blacklist {
		blacklist = append(blacklist, ip)
	}
	for _, net := range g.blacknets {
		blacklist = append(blacklist, net.String())
	}

	allowed := make([]string, 0, len(g.allowedGeo))
	for geo := range g.allowedGeo {
		allowed = append(allowed, geo)
	}

	return config.SecurityConfig{
		IPBlacklist:      blacklist,
		AllowedCountries: allowed,
		RateLimit:        g.config.Security.RateLimit,
	}
}

func (g *Gateway) UpdateConfig(cfg *config.Config) {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.config = cfg
	g.UpdateSecurity(cfg.Security)
	g.rebuildHandler()
}

func (g *Gateway) rebuildHandler() {
	// Base handler is the proxy logic
	base := http.HandlerFunc(g.proxyHandler)

	var middlewares []Middleware

	// Add Security Middlewares
	if g.config.Security.IPBlacklist != nil {
		middlewares = append(middlewares, g.IPBlacklistMiddleware())
	}

	if g.config.Security.RateLimit.Enabled {
		middlewares = append(middlewares, RateLimitMiddleware(
			g.config.Security.RateLimit.RPP,
			g.config.Security.RateLimit.Burst,
		))
	}

	// Always add GeoIP (handles Geo-fencing too)
	middlewares = append(middlewares, g.GeoIPMiddleware())

	g.handler = Chain(base, middlewares...)
}

func (g *Gateway) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	g.mu.RLock()
	handler := g.handler
	g.mu.RUnlock()

	handler.ServeHTTP(w, r)
}

func (g *Gateway) proxyHandler(w http.ResponseWriter, r *http.Request) {
	g.mu.RLock()
	defer g.mu.RUnlock()

	start := time.Now()
	path := r.URL.Path
	host := r.Host
	country := r.Header.Get("X-Geo-Country")

	var targetApi *config.ApiConfig

	// Try specific host + path matches first
	for i := range g.config.Products {
		p := &g.config.Products[i]
		for j := range p.Apis {
			a := &p.Apis[j]
			if a.Host != "" && matchHost(host, a.Host) && strings.HasPrefix(path, a.PathPrefix) {
				targetApi = a
				break
			}
		}
		if targetApi != nil {
			break
		}
	}

	// Fallback to path-only matches if no host match found
	if targetApi == nil {
		for i := range g.config.Products {
			p := &g.config.Products[i]
			for j := range p.Apis {
				a := &p.Apis[j]
				if a.Host == "" && strings.HasPrefix(path, a.PathPrefix) {
					targetApi = a
					break
				}
			}
			if targetApi != nil {
				break
			}
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
					// Also prioritize Host + Path in definitions
					if defs[i].Host != "" && matchHost(host, defs[i].Host) && strings.HasPrefix(path, defs[i].PathPrefix) {
						apiDef = &defs[i]
						break
					}
				}
				if apiDef == nil {
					for i := range defs {
						if defs[i].Host == "" && strings.HasPrefix(path, defs[i].PathPrefix) {
							apiDef = &defs[i]
							break
						}
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

	// Emit traffic packet for TUI Hub
	g.Hub.PublishTraffic(hub.TrafficEvent{
		Timestamp: start,
		Method:    r.Method,
		Path:      path,
		Backend:   backendName,
		Status:    rec.status,
		Latency:   elapsed,
		TenantID:  tenantID,
		Country:   country,
		IP:        r.RemoteAddr,
		Action:    "ALLOWED", // Default if it reached here
	})

	log.Printf("apim gateway: %s %s -> %s %d %dms", r.Method, path, backendName, rec.status, elapsed)
}

func matchHost(actual, target string) bool {
	// Strip port if present
	if h, _, err := net.SplitHostPort(actual); err == nil {
		actual = h
	}

	if actual == target {
		return true
	}
	if strings.HasPrefix(target, "*.") {
		suffix := target[1:] // suffix including the dot (e.g. ".example.com")
		if strings.HasSuffix(actual, suffix) {
			return true
		}
	}
	return false
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
