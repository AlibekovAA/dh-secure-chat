package db

import (
	"expvar"
	"time"

	"github.com/jackc/pgx/v4/pgxpool"
)

var (
	dbPoolAcquiredConns = expvar.NewInt("db_pool_acquired_connections")
	dbPoolIdleConns     = expvar.NewInt("db_pool_idle_connections")
	dbPoolMaxConns      = expvar.NewInt("db_pool_max_connections")
	dbPoolTotalConns    = expvar.NewInt("db_pool_total_connections")
)

func StartPoolMetrics(pool *pgxpool.Pool, interval time.Duration) {
	if interval <= 0 {
		interval = 30 * time.Second
	}

	ticker := time.NewTicker(interval)
	go func() {
		for range ticker.C {
			stats := pool.Stat()
			dbPoolAcquiredConns.Set(int64(stats.AcquiredConns()))
			dbPoolIdleConns.Set(int64(stats.IdleConns()))
			dbPoolMaxConns.Set(int64(stats.MaxConns()))
			dbPoolTotalConns.Set(int64(stats.TotalConns()))
		}
	}()
}
