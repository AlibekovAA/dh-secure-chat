package db

import (
	"time"

	"github.com/jackc/pgx/v4/pgxpool"

	prommetrics "github.com/AlibekovAA/dh-secure-chat/backend/internal/common/prometheus"
)

func StartPoolMetrics(pool *pgxpool.Pool, interval time.Duration) {
	if interval <= 0 {
		interval = 30 * time.Second
	}

	ticker := time.NewTicker(interval)
	go func() {
		for range ticker.C {
			stats := pool.Stat()
			acquired := int64(stats.AcquiredConns())
			idle := int64(stats.IdleConns())
			max := int64(stats.MaxConns())
			total := int64(stats.TotalConns())

			prommetrics.DBPoolAcquiredConnections.Set(float64(acquired))
			prommetrics.DBPoolIdleConnections.Set(float64(idle))
			prommetrics.DBPoolMaxConnections.Set(float64(max))
			prommetrics.DBPoolTotalConnections.Set(float64(total))
		}
	}()
}
