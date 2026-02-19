package admin

import (
	"encoding/json"
	"net/http"

	"github.com/navantesolutions/apimcore/internal/store"
)

const keyPrefixLen = 8

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
	if len(prefix) > keyPrefixLen {
		prefix = prefix[:keyPrefixLen]
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
