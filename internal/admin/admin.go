package admin

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"github.com/navantesolutions/apimcore/internal/gateway"
	"github.com/navantesolutions/apimcore/internal/store"
)

type Handler struct {
	store   *store.Store
	prefix  string
	gateway *gateway.Gateway
}

func New(s *store.Store, prefix string, gw *gateway.Gateway) *Handler {
	return &Handler{store: s, prefix: prefix, gateway: gw}
}

func (h *Handler) Register(mux *http.ServeMux) {
	mux.HandleFunc(h.prefix+"/products", h.products)
	mux.HandleFunc(h.prefix+"/products/", h.productByID)
	mux.HandleFunc(h.prefix+"/definitions", h.definitions)
	mux.HandleFunc(h.prefix+"/definitions/", h.definitionByID)
	mux.HandleFunc(h.prefix+"/subscriptions", h.subscriptions)
	mux.HandleFunc(h.prefix+"/subscriptions/", h.subscriptionByID)
	mux.HandleFunc(h.prefix+"/keys", h.keys)
	mux.HandleFunc(h.prefix+"/keys/", h.keyByID)
	mux.HandleFunc(h.prefix+"/usage", h.usage)
	mux.HandleFunc(h.prefix+"/metrics/summary", h.metricsSummary)
}

func writeJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(v)
}

func idFromPath(path, pathPrefix string) int64 {
	rest := strings.TrimPrefix(path, pathPrefix)
	if rest == path {
		return 0
	}
	if idx := strings.Index(rest, "/"); idx >= 0 {
		rest = rest[:idx]
	}
	id, _ := strconv.ParseInt(rest, 10, 64)
	return id
}

func generateAPIKey() string {
	b := make([]byte, 24)
	_, _ = rand.Read(b)
	return "apim_" + hex.EncodeToString(b)
}

func hashKey(key string) string {
	h := sha256.Sum256([]byte(key))
	return hex.EncodeToString(h[:])
}
