package httpmetrics

import (
	"fmt"
	"net/http"
	"time"

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
		path := NormalizePath(r.URL.Path)
		service := c.prefix

		metrics.HTTPRequestsTotal.WithLabelValues(service, method, path).Inc()

		rec := &statusRecorder{ResponseWriter: w, status: http.StatusOK}
		next.ServeHTTP(rec, r)

		elapsed := time.Since(start)
		statusClass := fmt.Sprintf("%dxx", rec.status/100)

		metrics.HTTPRequestDurationSeconds.WithLabelValues(service, method, path, statusClass).Observe(elapsed.Seconds())
	})
}
