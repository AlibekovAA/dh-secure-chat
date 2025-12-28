package db

import (
	"time"

	"github.com/jackc/pgx/v4/pgxpool"

	"github.com/AlibekovAA/dh-secure-chat/backend/internal/common/constants"
	"github.com/AlibekovAA/dh-secure-chat/backend/internal/observability/metrics"
)

func StartPoolMetrics(pool *pgxpool.Pool, interval time.Duration) {
	if interval <= 0 {
		interval = constants.DBPoolMetricsInterval
	}

	ticker := time.NewTicker(interval)
	go func() {
		for range ticker.C {
			stats := pool.Stat()
			acquired := int64(stats.AcquiredConns())
			idle := int64(stats.IdleConns())
			max := int64(stats.MaxConns())
			total := int64(stats.TotalConns())

			metrics.DBPoolAcquiredConnections.Set(float64(acquired))
			metrics.DBPoolIdleConnections.Set(float64(idle))
			metrics.DBPoolMaxConnections.Set(float64(max))
			metrics.DBPoolTotalConnections.Set(float64(total))
		}
	}()
}
