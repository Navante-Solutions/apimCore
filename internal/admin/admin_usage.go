package admin

import (
	"net/http"
	"strconv"
	"time"
)

const defaultUsageHours = 24

func (h *Handler) usage(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	hours := defaultUsageHours
	if s := r.URL.Query().Get("hours"); s != "" {
		if v, err := strconv.Atoi(s); err == nil && v > 0 {
			hours = v
		}
	}
	since := time.Now().Add(-time.Duration(hours) * time.Hour)
	usage := h.store.UsageSince(since)
	writeJSON(w, map[string]any{"total": len(usage), "requests": usage})
}
