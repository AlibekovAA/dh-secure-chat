package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	DBPoolAcquiredConnections = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "db_pool_acquired_connections",
			Help: "Number of acquired database connections",
		},
	)

	DBPoolIdleConnections = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "db_pool_idle_connections",
			Help: "Number of idle database connections",
		},
	)

	DBPoolMaxConnections = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "db_pool_max_connections",
			Help: "Maximum number of database connections",
		},
	)

	DBPoolTotalConnections = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "db_pool_total_connections",
			Help: "Total number of database connections",
		},
	)

	DBQueryDurationSeconds = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "db_query_duration_seconds",
			Help:    "Duration of database queries in seconds",
			Buckets: []float64{0.001, 0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1, 2.5, 5},
		},
		[]string{"operation", "table"},
	)

	DBQueryErrors = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "db_query_errors_total",
			Help: "Total number of database query errors",
		},
		[]string{"operation", "table", "error_type"},
	)
)
