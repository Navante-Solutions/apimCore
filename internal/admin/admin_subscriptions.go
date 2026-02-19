package admin

import (
	"encoding/json"
	"net/http"

	"github.com/navantesolutions/apimcore/internal/store"
)

func (h *Handler) subscriptions(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		products := h.store.ListProducts()
		var list []store.Subscription
		for _, p := range products {
			list = append(list, h.store.ListSubscriptionsByProduct(p.ID)...)
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
