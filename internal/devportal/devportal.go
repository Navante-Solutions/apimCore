package devportal

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/navantesolutions/apimcore/internal/store"
)

type Handler struct {
	store  *store.Store
	prefix string
}

func New(s *store.Store, prefix string) *Handler {
	return &Handler{store: s, prefix: prefix}
}

func (h *Handler) Register(mux *http.ServeMux) {
	mux.HandleFunc(h.prefix+"/api/products", h.listProducts)
	mux.HandleFunc(h.prefix+"/api/apis", h.listAPIs)
	mux.HandleFunc(h.prefix+"/api/usage", h.usage)
	mux.HandleFunc(h.prefix+"/api/usage/subscription/", h.usageBySubscription)
}

func (h *Handler) listProducts(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	products := h.store.ListProducts()
	filtered := make([]store.ApiProduct, 0)
	for _, p := range products {
		if p.Published {
			filtered = append(filtered, p)
		}
	}
	writeJSON(w, filtered)
}

func (h *Handler) listAPIs(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	productID := r.URL.Query().Get("product_id")
	var result []store.ApiDefinition
	if productID == "" {
		products := h.store.ListProducts()
		for _, p := range products {
			if !p.Published {
				continue
			}
			defs := h.store.ListDefinitionsByProduct(p.ID)
			result = append(result, defs...)
		}
	} else {
		if id, err := strconv.ParseInt(productID, 10, 64); err == nil {
			result = h.store.ListDefinitionsByProduct(id)
		}
	}
	writeJSON(w, result)
}

func (h *Handler) usage(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	since := time.Now().Add(-24 * time.Hour)
	usage := h.store.UsageSince(since)
	type summary struct {
		Total  int            `json:"total"`
		ByPath map[string]int `json:"by_path"`
		ByApi  map[string]int `json:"by_api"`
	}
	byPath := make(map[string]int)
	byApi := make(map[string]int)
	for _, u := range usage {
		byPath[u.Path]++
		def := h.store.GetDefinition(u.ApiDefinitionID)
		name := "unknown"
		if def != nil {
			name = def.Name
		}
		byApi[name]++
	}
	writeJSON(w, summary{
		Total:  len(usage),
		ByPath: byPath,
		ByApi:  byApi,
	})
}

func (h *Handler) usageBySubscription(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	base := h.prefix + "/api/usage/subscription/"
	subIDStr := strings.TrimPrefix(r.URL.Path, base)
	subIDStr = strings.TrimSuffix(subIDStr, "/")
	if idx := strings.Index(subIDStr, "/"); idx >= 0 {
		subIDStr = subIDStr[:idx]
	}
	subID, err := strconv.ParseInt(subIDStr, 10, 64)
	if err != nil {
		http.Error(w, "invalid subscription id", http.StatusBadRequest)
		return
	}
	since := time.Now().Add(-24 * time.Hour)
	usage := h.store.UsageBySubscription(subID, since)
	writeJSON(w, map[string]any{"total": len(usage), "requests": usage})
}

func writeJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(v)
}
