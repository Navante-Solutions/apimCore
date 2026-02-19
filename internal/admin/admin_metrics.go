package admin

import (
	"net/http"
	"strconv"
	"time"
)

const defaultMetricsHours = 1

func (h *Handler) metricsSummary(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	hours := defaultMetricsHours
	if s := r.URL.Query().Get("hours"); s != "" {
		if v, err := strconv.Atoi(s); err == nil && v > 0 {
			hours = v
		}
	}
	since := time.Now().Add(-time.Duration(hours) * time.Hour)

	p95Ms, _ := h.store.PercentileResponseTimeMsSince(since, 0.95)
	p99Ms, _ := h.store.PercentileResponseTimeMsSince(since, 0.99)
	errorRate, totalReqs, errorsCount := h.store.ErrorRateSince(since)
	rpsByRoute := h.store.RPSByRouteSince(since)
	usageByVersion := h.store.UsageByVersionSince(since)
	avgBackendMs, avgGatewayMs, backendCount := h.store.AvgBackendVsGatewaySince(since)

	byTenant := make(map[string]int64)
	for _, tenantID := range h.store.UniqueTenantIDs() {
		usage := h.store.UsageByTenant(tenantID, since)
		byTenant[tenantID] = int64(len(usage))
	}

	var rateLimitHits int64
	if h.gateway != nil {
		_, rateLimitHits = h.gateway.Stats()
	}

	writeJSON(w, map[string]any{
		"window_hours":    hours,
		"since":           since.Format(time.RFC3339),
		"latency_p95_ms":  p95Ms,
		"latency_p99_ms":  p99Ms,
		"error_rate":      errorRate,
		"total_requests":  totalReqs,
		"error_requests":  errorsCount,
		"rps_by_route":    rpsByRoute,
		"rate_limit_hits": rateLimitHits,
		"usage_by_tenant": byTenant,
		"usage_by_version": usageByVersion,
		"backend_vs_gateway": map[string]any{
			"avg_backend_ms":  avgBackendMs,
			"avg_gateway_ms":  avgGatewayMs,
			"requests_with_backend_latency": backendCount,
		},
	})
}
