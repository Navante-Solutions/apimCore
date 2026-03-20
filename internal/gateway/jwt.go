package gateway

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/MicahParks/keyfunc/v3"
)

// JWTMiddleware validates Keycloak JWTs.
func (g *Gateway) JWTMiddleware() Middleware {
	// Simple cache for JWKS to avoid fetching on every request
	// In a production app, this should be configuration-driven
	var kf keyfunc.Keyfunc
	var lastFetch time.Time

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			authHeader := r.Header.Get("Authorization")
			if authHeader == "" || !strings.HasPrefix(authHeader, "Bearer ") {
				// If no JWT, fall back to API Key or reject if required
				// For now, we allow fallback to API Key if present
				if r.Header.Get(HeaderAPIKey) != "" {
					next.ServeHTTP(w, r)
					return
				}
				http.Error(w, "Unauthorized: Missing Token", http.StatusUnauthorized)
				return
			}

			tokenString := strings.TrimPrefix(authHeader, "Bearer ")

			// Resolve JWKS dynamically from issuer if not cached or old
			// This is a simplified version. Production should handle multiple issuers correctly.
			// IdentityCore uses iss + "/protocol/openid-connect/certs"
			
			// For simplicity in this implementation, we'll try to get the 'iss' from the token first
			parser := jwt.NewParser()
			token, _, err := parser.ParseUnverified(tokenString, jwt.MapClaims{})
			if err != nil {
				http.Error(w, "Unauthorized: Invalid Token Format", http.StatusUnauthorized)
				return
			}

			claims, ok := token.Claims.(jwt.MapClaims)
			if !ok {
				http.Error(w, "Unauthorized: Invalid Claims", http.StatusUnauthorized)
				return
			}

			iss, ok := claims["iss"].(string)
			if !ok {
				http.Error(w, "Unauthorized: Missing Issuer", http.StatusUnauthorized)
				return
			}

			// Initialize or refresh Keyfunc if needed
			if kf == nil || time.Since(lastFetch) > 1*time.Hour {
				jwksURL := fmt.Sprintf("%s/protocol/openid-connect/certs", iss)
				newKf, err := keyfunc.NewDefault([]string{jwksURL})
				if err != nil {
					http.Error(w, "Internal Server Error: JWKS fetch failed", http.StatusInternalServerError)
					return
				}
				kf = newKf
				lastFetch = time.Now()
			}

			// Validate JWT
			validatedToken, err := jwt.Parse(tokenString, kf.Keyfunc)
			if err != nil || !validatedToken.Valid {
				http.Error(w, "Unauthorized: Token Validation Failed", http.StatusUnauthorized)
				return
			}

			// Extract tenant_id and inject into header if present
			if tenantID, ok := claims["tenant_id"].(string); ok {
				r.Header.Set(HeaderTenantID, tenantID)
			}

			next.ServeHTTP(w, r)
		})
	}
}
