package gateway

import (
	"net"
	"net/http"
	"sync/atomic"
	"time"

	"github.com/navantesolutions/apimcore/internal/hub"
)

// IPBlacklistMiddleware blocks requests from IPs in the blacklist.
func (g *Gateway) IPBlacklistMiddleware() Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			g.securityMu.Lock()
			remoteIP, _, _ := net.SplitHostPort(r.RemoteAddr)
			ip := net.ParseIP(remoteIP)

			blocked := g.blacklist[remoteIP]
			if !blocked {
				for _, ipnet := range g.blacknets {
					if ipnet.Contains(ip) {
						blocked = true
						break
					}
				}
			}
			g.securityMu.Unlock()

			if blocked {
				atomic.AddInt64(&g.blockedCount, 1)
				if g.Hub != nil {
					g.Hub.PublishTraffic(hub.TrafficEvent{
						Timestamp: time.Now(),
						Method:    r.Method,
						Path:      r.URL.Path,
						Backend:   "",
						Status:    http.StatusForbidden,
						Latency:   0,
						TenantID:  "",
						Country:   "",
						IP:        remoteIP,
						Action:    "BLOCKED",
					})
				}
				http.Error(w, "Forbidden: IP Blacklisted", http.StatusForbidden)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// GeoIPMiddleware mocks GeoIP resolution and implements Geo-fencing.
func (g *Gateway) GeoIPMiddleware() Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			remoteIP, _, _ := net.SplitHostPort(r.RemoteAddr)
			var country string
			if remoteIP == "127.0.0.1" || remoteIP == "::1" {
				country = "Local"
			} else if remoteIP == "8.8.8.8" {
				country = "US"
			} else {
				if len(remoteIP)%2 == 0 {
					country = "BR"
				} else {
					country = "DE"
				}
			}

			r.Header.Set("X-Geo-Country", country)

			g.securityMu.Lock()
			allowed := len(g.allowedGeo) == 0 || g.allowedGeo[country]
			g.securityMu.Unlock()

			if !allowed {
				atomic.AddInt64(&g.blockedCount, 1)
				if g.Hub != nil {
					g.Hub.PublishTraffic(hub.TrafficEvent{
						Timestamp: time.Now(),
						Method:    r.Method,
						Path:      r.URL.Path,
						Backend:   "",
						Status:    http.StatusForbidden,
						Latency:   0,
						TenantID:  "",
						Country:   country,
						IP:        remoteIP,
						Action:    "BLOCKED",
					})
				}
				http.Error(w, "Forbidden: Geo-fenced", http.StatusForbidden)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}
