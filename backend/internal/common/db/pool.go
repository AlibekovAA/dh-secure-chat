package db

import (
	"context"
	"time"

	"github.com/jackc/pgx/v4/pgxpool"

	"github.com/AlibekovAA/dh-secure-chat/backend/internal/common/constants"
	"github.com/AlibekovAA/dh-secure-chat/backend/internal/common/logger"
)

type PoolConfig struct {
	MaxOpenConns    int
	MinOpenConns    int
	ConnMaxLifetime time.Duration
	ConnMaxIdleTime time.Duration
	HealthCheck     time.Duration
	ConnectTimeout  time.Duration
	MaxAttempts     int
	RetryDelay      time.Duration
}

func DefaultPoolConfig() PoolConfig {
	return PoolConfig{
		MaxOpenConns:    constants.DBPoolMaxOpenConns,
		MinOpenConns:    constants.DBPoolMinOpenConns,
		ConnMaxLifetime: constants.DBPoolConnMaxLifetime,
		ConnMaxIdleTime: constants.DBPoolConnMaxIdleTime,
		HealthCheck:     constants.DBPoolHealthCheck,
		ConnectTimeout:  constants.DBPoolConnectTimeout,
		MaxAttempts:     constants.DBPoolMaxAttempts,
		RetryDelay:      constants.DBPoolRetryDelay,
	}
}

func NewPool(log *logger.Logger, databaseURL string) *pgxpool.Pool {
	return NewPoolWithConfig(context.Background(), log, databaseURL, DefaultPoolConfig())
}

func NewPoolWithConfig(ctx context.Context, log *logger.Logger, databaseURL string, cfg PoolConfig) *pgxpool.Pool {
	pgxCfg, err := pgxpool.ParseConfig(databaseURL)
	if err != nil {
		log.Fatalf("failed to parse database url: %v", err)
	}

	pgxCfg.MaxConns = int32(cfg.MaxOpenConns)
	pgxCfg.MinConns = int32(cfg.MinOpenConns)
	pgxCfg.MaxConnLifetime = cfg.ConnMaxLifetime
	pgxCfg.MaxConnIdleTime = cfg.ConnMaxIdleTime
	pgxCfg.HealthCheckPeriod = cfg.HealthCheck
	pgxCfg.ConnConfig.ConnectTimeout = cfg.ConnectTimeout
	pgxCfg.ConnConfig.RuntimeParams = map[string]string{
		"application_name": "dh-secure-chat",
	}

	if cfg.MaxAttempts <= 0 {
		cfg.MaxAttempts = 1
	}
	if cfg.RetryDelay <= 0 {
		cfg.RetryDelay = time.Second
	}

	for attempt := 1; attempt <= cfg.MaxAttempts; attempt++ {
		pool, err := pgxpool.ConnectConfig(ctx, pgxCfg)
		if err == nil {
			log.Infof("database connection pool initialized: max=%d, min=%d", pgxCfg.MaxConns, pgxCfg.MinConns)
			return pool
		}

		log.Warnf("failed to connect to database (attempt %d/%d): %v", attempt, cfg.MaxAttempts, err)

		if attempt == cfg.MaxAttempts {
			log.Fatalf("failed to connect to database after %d attempts: %v", cfg.MaxAttempts, err)
			return nil
		}

		select {
		case <-ctx.Done():
			log.Fatalf("database connection interrupted: %v", ctx.Err())
			return nil
		case <-time.After(cfg.RetryDelay):
		}
	}

	log.Fatalf("failed to connect to database after %d attempts", cfg.MaxAttempts)
	return nil
}
