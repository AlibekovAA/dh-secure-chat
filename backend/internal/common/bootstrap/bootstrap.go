package bootstrap

import (
	"os"

	"github.com/jackc/pgx/v4/pgxpool"

	"github.com/AlibekovAA/dh-secure-chat/backend/internal/common/db"
	"github.com/AlibekovAA/dh-secure-chat/backend/internal/common/logger"
)

type AppDependencies struct {
	Log  *logger.Logger
	Pool *pgxpool.Pool
}

func InitializeLogger(serviceName string) (*logger.Logger, error) {
	return logger.New(os.Getenv("LOG_DIR"), serviceName, os.Getenv("LOG_LEVEL"))
}

func InitializeDatabase(log *logger.Logger, databaseURL string) (*pgxpool.Pool, error) {
	pool := db.NewPool(log, databaseURL)
	return pool, nil
}
