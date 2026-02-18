package admin

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
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
	mux.HandleFunc(h.prefix+"/products", h.products)
	mux.HandleFunc(h.prefix+"/products/", h.productByID)
	mux.HandleFunc(h.prefix+"/definitions", h.definitions)
	mux.HandleFunc(h.prefix+"/definitions/", h.definitionByID)
	mux.HandleFunc(h.prefix+"/subscriptions", h.subscriptions)
	mux.HandleFunc(h.prefix+"/subscriptions/", h.subscriptionByID)
	mux.HandleFunc(h.prefix+"/keys", h.keys)
	mux.HandleFunc(h.prefix+"/keys/", h.keyByID)
	mux.HandleFunc(h.prefix+"/usage", h.usage)
}

func (h *Handler) products(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		list := h.store.ListProducts()
		writeJSON(w, list)
	case http.MethodPost:
		var p store.ApiProduct
		if err := json.NewDecoder(r.Body).Decode(&p); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		id := h.store.CreateProduct(&p)
		p.ID = id
		w.WriteHeader(http.StatusCreated)
		writeJSON(w, p)
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func (h *Handler) productByID(w http.ResponseWriter, r *http.Request) {
	id := idFromPath(r.URL.Path, h.prefix+"/products/")
	if id == 0 {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}
	if r.Method == http.MethodGet {
		p := h.store.GetProduct(id)
		if p == nil {
			http.Error(w, "not found", http.StatusNotFound)
			return
		}
		writeJSON(w, p)
		return
	}
	http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
}

func (h *Handler) definitions(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var d store.ApiDefinition
	if err := json.NewDecoder(r.Body).Decode(&d); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	id := h.store.CreateDefinition(&d)
	d.ID = id
	w.WriteHeader(http.StatusCreated)
	writeJSON(w, d)
}

func (h *Handler) definitionByID(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	id := idFromPath(r.URL.Path, h.prefix+"/definitions/")
	if id == 0 {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}
	d := h.store.GetDefinition(id)
	if d == nil {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}
	writeJSON(w, d)
}

func (h *Handler) subscriptions(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		products := h.store.ListProducts()
		var list []store.Subscription
		for _, p := range products {
			subs := h.store.ListSubscriptionsByProduct(p.ID)
			list = append(list, subs...)
		}
		writeJSON(w, list)
	case http.MethodPost:
		var sub store.Subscription
		if err := json.NewDecoder(r.Body).Decode(&sub); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		id := h.store.CreateSubscription(&sub)
		sub.ID = id
		w.WriteHeader(http.StatusCreated)
		writeJSON(w, sub)
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func (h *Handler) subscriptionByID(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	id := idFromPath(r.URL.Path, h.prefix+"/subscriptions/")
	if id == 0 {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}
	sub := h.store.GetSubscription(id)
	if sub == nil {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}
	writeJSON(w, sub)
}

func (h *Handler) keys(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var req struct {
		SubscriptionID int64  `json:"subscription_id"`
		Name           string `json:"name"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	rawKey := generateAPIKey()
	hash := hashKey(rawKey)
	prefix := rawKey
	if len(prefix) > 8 {
		prefix = prefix[:8]
	}
	k := &store.ApiKey{
		SubscriptionID: int64(req.SubscriptionID),
		KeyHash:        hash,
		KeyPrefix:      prefix,
		Name:           req.Name,
		Active:         true,
	}
	id := h.store.CreateApiKey(k)
	k.ID = id
	w.WriteHeader(http.StatusCreated)
	writeJSON(w, map[string]any{
		"id":         id,
		"key":        rawKey,
		"prefix":     prefix,
		"name":       k.Name,
		"created_at": k.CreatedAt,
	})
}

func (h *Handler) keyByID(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	id := idFromPath(r.URL.Path, h.prefix+"/keys/")
	if id == 0 {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}
	k := h.store.GetKeyByID(id)
	if k == nil {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}
	writeJSON(w, map[string]any{
		"id": k.ID, "subscription_id": k.SubscriptionID, "key_prefix": k.KeyPrefix,
		"name": k.Name, "active": k.Active, "created_at": k.CreatedAt, "last_used_at": k.LastUsedAt,
	})
}

func (h *Handler) usage(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	hours := 24
	if s := r.URL.Query().Get("hours"); s != "" {
		if v, err := strconv.Atoi(s); err == nil && v > 0 {
			hours = v
		}
	}
	since := time.Now().Add(-time.Duration(hours) * time.Hour)
	usage := h.store.UsageSince(since)
	writeJSON(w, map[string]any{"total": len(usage), "requests": usage})
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
