package db

import (
	"context"
	"time"

	"github.com/jackc/pgx/v4/pgxpool"

	"github.com/AlibekovAA/dh-secure-chat/backend/internal/common/logger"
)

func NewPool(databaseURL string) *pgxpool.Pool {
	log := logger.GetInstance()

	cfg, err := pgxpool.ParseConfig(databaseURL)
	if err != nil {
		log.Fatalf("failed to parse database url: %v", err)
	}

	const maxAttempts = 10
	const delay = time.Second

	for attempt := 1; attempt <= maxAttempts; attempt++ {
		pool, err := pgxpool.ConnectConfig(context.Background(), cfg)
		if err == nil {
			return pool
		}

		log.Warnf("failed to connect to database (attempt %d/%d): %v", attempt, maxAttempts, err)

		if attempt == maxAttempts {
			log.Fatalf("failed to connect to database after %d attempts: %v", maxAttempts, err)
		}

		time.Sleep(delay)
	}

	return nil
}
