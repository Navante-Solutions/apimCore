package admin

import (
	"encoding/json"
	"net/http"

	"github.com/navantesolutions/apimcore/internal/store"
)

func (h *Handler) products(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		writeJSON(w, h.store.ListProducts())
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
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	p := h.store.GetProduct(id)
	if p == nil {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}
	writeJSON(w, p)
}
