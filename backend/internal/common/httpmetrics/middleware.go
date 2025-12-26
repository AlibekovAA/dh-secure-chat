package httpmetrics

import (
	"fmt"
	"net/http"
	"time"

	"github.com/prometheus/client_golang/prometheus"

	"github.com/AlibekovAA/dh-secure-chat/backend/internal/observability/metrics"
)

type Collector struct {
	prefix string
}

type statusRecorder struct {
	http.ResponseWriter
	status int
}

func (r *statusRecorder) WriteHeader(code int) {
	r.status = code
	r.ResponseWriter.WriteHeader(code)
}

func New(prefix string) *Collector {
	return &Collector{
		prefix: prefix,
	}
}

func (c *Collector) Wrap(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		method := r.Method
		path := r.URL.Path

		var promRequestsTotal *prometheus.CounterVec
		var promRequestsInFlight prometheus.Gauge
		var promRequestDuration *prometheus.HistogramVec

		if c.prefix == "auth" {
			promRequestsTotal = metrics.AuthRequestsTotal
			promRequestsInFlight = metrics.AuthRequestsInFlight
			promRequestDuration = metrics.AuthRequestDurationSeconds
		} else if c.prefix == "chat" {
			promRequestsTotal = metrics.ChatRequestsTotal
			promRequestsInFlight = metrics.ChatRequestsInFlight
			promRequestDuration = metrics.ChatRequestDurationSeconds
		}

		if promRequestsTotal != nil {
			promRequestsTotal.WithLabelValues(method, path).Inc()
		}
		if promRequestsInFlight != nil {
			promRequestsInFlight.Inc()
		}

		rec := &statusRecorder{ResponseWriter: w, status: http.StatusOK}
		next.ServeHTTP(rec, r)

		elapsed := time.Since(start)
		statusClass := fmt.Sprintf("%dxx", rec.status/100)

		if promRequestsInFlight != nil {
			promRequestsInFlight.Dec()
		}
		if promRequestDuration != nil {
			promRequestDuration.WithLabelValues(method, path, statusClass).Observe(elapsed.Seconds())
		}
	})
}
