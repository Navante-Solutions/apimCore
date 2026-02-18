package gateway

import (
	"net"
	"net/http"
	"sync"

	"golang.org/x/time/rate"
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
				http.Error(w, "Forbidden: IP Blacklisted", http.StatusForbidden)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// RateLimitMiddleware provides Anti-DDoS protection using a token bucket.
func RateLimitMiddleware(rps float64, burst int) Middleware {
	var mu sync.Mutex
	limiters := make(map[string]*rate.Limiter)

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			mu.Lock()
			remoteIP, _, _ := net.SplitHostPort(r.RemoteAddr)
			limiter, ok := limiters[remoteIP]
			if !ok {
				limiter = rate.NewLimiter(rate.Limit(rps), burst)
				limiters[remoteIP] = limiter
			}
			mu.Unlock()

			if !limiter.Allow() {
				http.Error(w, "Too Many Requests", http.StatusTooManyRequests)
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
			// Mock logic: local IPs are "Home", 8.8.8.8 is "US", others are "Unknown"
			remoteIP, _, _ := net.SplitHostPort(r.RemoteAddr)
			country := "Unknown"

			if remoteIP == "127.0.0.1" || remoteIP == "::1" {
				country = "Local"
			} else if remoteIP == "8.8.8.8" {
				country = "US"
			} else {
				// Random-ish country for demo
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
				http.Error(w, "Forbidden: Geo-fenced", http.StatusForbidden)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}
