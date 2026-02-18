package meter

import (
	"time"

	"github.com/prometheus/client_golang/prometheus"

	"github.com/navantesolutions/apimcore/internal/store"
)

type Meter struct {
	store      *store.Store
	requestCnt *prometheus.CounterVec
	requestLat *prometheus.HistogramVec
	usageTotal prometheus.Counter
}

func New(s *store.Store, reg prometheus.Registerer) *Meter {
	requestCnt := prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "apim_requests_total",
			Help: "Total API requests through the gateway",
		},
		[]string{"backend", "method", "path_prefix", "status"},
	)
	requestLat := prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "apim_request_duration_seconds",
			Help:    "Request latency in seconds",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"backend", "path_prefix"},
	)
	usageTotal := prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: "apim_usage_records_total",
			Help: "Total usage records stored",
		},
	)
	if reg != nil {
		reg.MustRegister(requestCnt, requestLat, usageTotal)
	}
	return &Meter{
		store:      s,
		requestCnt: requestCnt,
		requestLat: requestLat,
		usageTotal: usageTotal,
	}
}

func (m *Meter) Record(backend, pathPrefix, method string, status int, durationMs int64, subID, apiDefID int64, tenantID string) {
	m.requestCnt.WithLabelValues(backend, method, pathPrefix, statusLabel(status)).Inc()
	m.requestLat.WithLabelValues(backend, pathPrefix).Observe(float64(durationMs) / 1000.0)
	m.store.RecordUsage(store.RequestUsage{
		SubscriptionID:  subID,
		ApiDefinitionID: apiDefID,
		TenantID:        tenantID,
		Method:          method,
		Path:            pathPrefix,
		StatusCode:      status,
		ResponseTimeMs:  durationMs,
	})
	m.usageTotal.Inc()
}

func statusLabel(code int) string {
	if code >= 200 && code < 300 {
		return "2xx"
	}
	if code >= 400 && code < 500 {
		return "4xx"
	}
	if code >= 500 {
		return "5xx"
	}
	return "other"
}

func (m *Meter) StatsSince(since time.Time) (total int64, byBackend map[string]int64, byPath map[string]int64) {
	usage := m.store.UsageSince(since)
	total = int64(len(usage))
	byBackend = make(map[string]int64)
	byPath = make(map[string]int64)
	for _, u := range usage {
		def := m.store.GetDefinition(u.ApiDefinitionID)
		if def != nil {
			byBackend[def.Name]++
			byPath[def.PathPrefix]++
		}
	}
	return total, byBackend, byPath
}
