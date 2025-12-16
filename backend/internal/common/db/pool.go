package db

import (
	"context"
	"time"

	"github.com/jackc/pgx/v4/pgxpool"

	"github.com/AlibekovAA/dh-secure-chat/backend/internal/common/logger"
)

func NewPool(log *logger.Logger, databaseURL string) *pgxpool.Pool {
	cfg, err := pgxpool.ParseConfig(databaseURL)
	if err != nil {
		log.Fatalf("failed to parse database url: %v", err)
	}

	cfg.MaxConns = 25
	cfg.MinConns = 5
	cfg.MaxConnLifetime = time.Hour
	cfg.MaxConnIdleTime = 30 * time.Minute
	cfg.HealthCheckPeriod = time.Minute

	const maxAttempts = 10
	const delay = time.Second

	for attempt := 1; attempt <= maxAttempts; attempt++ {
		pool, err := pgxpool.ConnectConfig(context.Background(), cfg)
		if err == nil {
			log.Infof("database connection pool initialized: max=%d, min=%d", cfg.MaxConns, cfg.MinConns)
			return pool
		}

		log.Warnf("failed to connect to database (attempt %d/%d): %v", attempt, maxAttempts, err)

		if attempt == maxAttempts {
			log.Fatalf("failed to connect to database after %d attempts: %v", maxAttempts, err)
		}

		time.Sleep(delay)
	}

	panic("unreachable code: failed to connect to database after all attempts")
}
