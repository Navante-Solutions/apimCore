package admin

import (
	"encoding/json"
	"net/http"

	"github.com/navantesolutions/apimcore/internal/store"
)

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
