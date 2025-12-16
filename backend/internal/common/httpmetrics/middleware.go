package httpmetrics

import (
	"expvar"
	"fmt"
	"net/http"
	"time"
)

type Collector struct {
	totalRequests      *expvar.Int
	inFlightRequests   *expvar.Int
	requestDurationMs  *expvar.Map
	lastRequestStatus  *expvar.Map
	lastRequestSeconds *expvar.Map
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
		totalRequests:      expvar.NewInt(prefix + "_requests_total"),
		inFlightRequests:   expvar.NewInt(prefix + "_requests_in_flight"),
		requestDurationMs:  expvar.NewMap(prefix + "_request_duration_ms"),
		lastRequestStatus:  expvar.NewMap(prefix + "_last_status_by_path"),
		lastRequestSeconds: expvar.NewMap(prefix + "_last_duration_seconds_by_path"),
	}
}

func (c *Collector) Wrap(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		c.inFlightRequests.Add(1)
		c.totalRequests.Add(1)

		pathKey := r.Method + " " + r.URL.Path

		rec := &statusRecorder{ResponseWriter: w, status: http.StatusOK}
		next.ServeHTTP(rec, r)

		c.inFlightRequests.Add(-1)

		elapsed := time.Since(start)
		statusClass := fmt.Sprintf("%dxx", rec.status/100)
		c.requestDurationMs.Add(pathKey, elapsed.Milliseconds())
		c.lastRequestStatus.Set(pathKey, expvar.Func(func() any { return statusClass }))
		c.lastRequestSeconds.Set(pathKey, expvar.Func(func() any { return elapsed.Seconds() }))
	})
}
