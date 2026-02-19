package gateway

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"log"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/navantesolutions/apimcore/config"
	"github.com/navantesolutions/apimcore/internal/hub"
	"github.com/navantesolutions/apimcore/internal/meter"
	"github.com/navantesolutions/apimcore/internal/store"
	"golang.org/x/time/rate"
)

type backendLatencyKey struct{}

type backendLatencyHolder struct {
	Ms int64
}

type timeoutTransport struct {
	base    http.RoundTripper
	timeout time.Duration
}

func (t *timeoutTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	if t.timeout > 0 {
		ctx, cancel := context.WithTimeout(req.Context(), t.timeout)
		defer cancel()
		req = req.WithContext(ctx)
	}
	return t.base.RoundTrip(req)
}

type measuringTransport struct {
	base http.RoundTripper
}

func (t *measuringTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	start := time.Now()
	resp, err := t.base.RoundTrip(req)
	if holder, _ := req.Context().Value(backendLatencyKey{}).(*backendLatencyHolder); holder != nil {
		holder.Ms = time.Since(start).Milliseconds()
	}
	return resp, err
}

const (
	HeaderAPIKey    = "X-Api-Key"
	HeaderTenantID  = "X-Tenant-Id"
	HeaderRequestID = "X-Request-Id"
	HeaderGeoCountry = "X-Geo-Country"
	KeyPrefixLen     = 8
	RateLimiterMapMaxSize = 100_000
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

func trafficEventFromRequest(r *http.Request, ts time.Time, action string, status int, totalMs, backendMs int64, backend, tenantID, remoteIP string) hub.TrafficEvent {
	if remoteIP == "" {
		remoteIP = r.RemoteAddr
	}
	return hub.TrafficEvent{
		Timestamp:      ts,
		Method:         r.Method,
		Path:           r.URL.Path,
		Backend:        backend,
		Status:         status,
		Latency:        totalMs,
		BackendLatency: backendMs,
		TenantID:       tenantID,
		Country:        r.Header.Get(HeaderGeoCountry),
		IP:             remoteIP,
		Action:         action,
	}
}

type Gateway struct {
	mu               sync.RWMutex
	config           *config.Config
	store            *store.Store
	meter            *meter.Meter
	proxy            *httputil.ReverseProxy
	handler          http.Handler
	Hub              *hub.Broadcaster
	securityMu       sync.Mutex
	blacklist        map[string]bool
	blacknets        []*net.IPNet
	allowedGeo       map[string]bool
	blockedCount     int64
	rateLimitedCount int64
}

func New(cfg *config.Config, s *store.Store, m *meter.Meter, h *hub.Broadcaster) *Gateway {
	g := &Gateway{
		config: cfg,
		store:  s,
		meter:  m,
		proxy:  &httputil.ReverseProxy{},
		Hub:    h,
	}
	timeoutTrans := &timeoutTransport{
		base:    http.DefaultTransport,
		timeout: time.Duration(cfg.Gateway.BackendTimeoutSeconds) * time.Second,
	}
	g.proxy.Transport = &measuringTransport{base: timeoutTrans}
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

func (g *Gateway) Stats() (blocked, rateLimited int64) {
	return atomic.LoadInt64(&g.blockedCount), atomic.LoadInt64(&g.rateLimitedCount)
}

func (g *Gateway) UpdateConfig(cfg *config.Config) {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.config = cfg
	if g.proxy.Transport != nil {
		if tt, ok := g.proxy.Transport.(*timeoutTransport); ok {
			tt.timeout = time.Duration(cfg.Gateway.BackendTimeoutSeconds) * time.Second
		} else {
			g.proxy.Transport = &timeoutTransport{
				base:    http.DefaultTransport,
				timeout: time.Duration(cfg.Gateway.BackendTimeoutSeconds) * time.Second,
			}
		}
	}
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
		middlewares = append(middlewares, g.RateLimitMiddleware())
	}

	// Always add GeoIP (handles Geo-fencing too)
	middlewares = append(middlewares, g.GeoIPMiddleware())

	g.handler = Chain(base, middlewares...)
}

func (g *Gateway) RateLimitMiddleware() Middleware {
	var mu sync.Mutex
	limiters := make(map[string]*rate.Limiter)
	rps := g.config.Security.RateLimit.RPS
	burst := g.config.Security.RateLimit.Burst
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			mu.Lock()
			remoteIP, _, _ := net.SplitHostPort(r.RemoteAddr)
			if len(limiters) >= RateLimiterMapMaxSize {
				limiters = make(map[string]*rate.Limiter)
			}
			limiter, ok := limiters[remoteIP]
			if !ok {
				limiter = rate.NewLimiter(rate.Limit(rps), burst)
				limiters[remoteIP] = limiter
			}
			mu.Unlock()
			if !limiter.Allow() {
				atomic.AddInt64(&g.rateLimitedCount, 1)
				g.meter.IncrementRateLimit()
				if g.Hub != nil {
					g.Hub.PublishTraffic(trafficEventFromRequest(r, time.Now(), "RATE_LIMIT", http.StatusTooManyRequests, 0, 0, "", "", remoteIP))
				}
				http.Error(w, "Too Many Requests", http.StatusTooManyRequests)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
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
	targetApi, apiDef, sub := g.resolveRoute(host, path, r.Header.Get(HeaderAPIKey))
	if targetApi == nil {
		http.Error(w, "no route for path", http.StatusNotFound)
		g.meter.Record("", "", r.Method, http.StatusNotFound, time.Since(start).Milliseconds(), 0, 0, 0, "")
		return
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
		g.meter.Record(backendName, targetApi.PathPrefix, r.Method, 502, time.Since(start).Milliseconds(), 0, 0, apiDefID, "")
		return
	}

	rec := &responseRecorder{ResponseWriter: w, status: 200}
	if sub != nil && sub.TenantID != "" {
		r.Header.Set(HeaderTenantID, sub.TenantID)
	}

	var addHeaders map[string]string
	var pathPrefixToStrip string
	var stripPath bool
	if apiDef != nil {
		addHeaders = apiDef.AddHeaders
		pathPrefixToStrip = apiDef.PathPrefix
		stripPath = apiDef.StripPathPrefix
	} else {
		addHeaders = targetApi.AddHeaders
		pathPrefixToStrip = targetApi.PathPrefix
		stripPath = targetApi.StripPathPrefix
	}
	for k, v := range addHeaders {
		r.Header.Set(k, v)
	}

	dest := *targetURL
	g.proxy.Director = func(req *http.Request) {
		if stripPath && pathPrefixToStrip != "" {
			req.URL.Path = strings.TrimPrefix(req.URL.Path, pathPrefixToStrip)
			if req.URL.Path == "" {
				req.URL.Path = "/"
			}
		}
		req.URL.Scheme = dest.Scheme
		req.URL.Host = dest.Host
		req.Host = dest.Host
	}
	holder := &backendLatencyHolder{}
	r = r.WithContext(context.WithValue(r.Context(), backendLatencyKey{}, holder))
	g.proxy.ServeHTTP(rec, r)

	elapsed := time.Since(start).Milliseconds()
	backendMs := holder.Ms
	subID := int64(0)
	tenantID := ""
	if sub != nil {
		subID = sub.ID
		tenantID = sub.TenantID
	}
	g.meter.Record(backendName, targetApi.PathPrefix, r.Method, rec.status, elapsed, backendMs, subID, apiDefID, tenantID)

	if g.Hub != nil {
		g.Hub.PublishTraffic(trafficEventFromRequest(r, start, "ALLOWED", rec.status, elapsed, backendMs, backendName, tenantID, ""))
	}

	log.Printf("apimcore gateway: %s %s -> %s %d %dms", r.Method, path, backendName, rec.status, elapsed)
}

func (g *Gateway) resolveRoute(host, path, apiKey string) (targetApi *config.ApiConfig, apiDef *store.ApiDefinition, sub *store.Subscription) {
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
		return nil, nil, nil
	}

	if apiKey != "" {
		hash := hashKey(apiKey)
		prefix := apiKey
		if len(prefix) > KeyPrefixLen {
			prefix = prefix[:KeyPrefixLen]
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
	return targetApi, apiDef, sub
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
