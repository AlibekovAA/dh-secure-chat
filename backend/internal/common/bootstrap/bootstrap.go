package bootstrap

import (
	"fmt"
	"os"

	"github.com/jackc/pgx/v4/pgxpool"

	"github.com/AlibekovAA/dh-secure-chat/backend/internal/common/config"
	"github.com/AlibekovAA/dh-secure-chat/backend/internal/common/constants"
	"github.com/AlibekovAA/dh-secure-chat/backend/internal/common/db"
	"github.com/AlibekovAA/dh-secure-chat/backend/internal/common/logger"
	identityrepo "github.com/AlibekovAA/dh-secure-chat/backend/internal/identity/repository"
	identityservice "github.com/AlibekovAA/dh-secure-chat/backend/internal/identity/service"
	userrepo "github.com/AlibekovAA/dh-secure-chat/backend/internal/user/repository"
)

type App struct {
	Log             *logger.Logger
	Pool            *pgxpool.Pool
	UserRepo        userrepo.Repository
	IdentityRepo    identityrepo.Repository
	IdentityService *identityservice.IdentityService
}

type AuthApp struct {
	App
	Config config.AuthConfig
}

type ChatApp struct {
	App
	Config config.ChatConfig
}

func NewAuthApp() (*AuthApp, error) {
	log, err := initializeLogger("auth")
	if err != nil {
		return nil, fmt.Errorf("failed to initialize logger: %w", err)
	}

	cfg, err := config.LoadAuthConfig()
	if err != nil {
		log.Fatalf("failed to load config: %v", err)
		return nil, err
	}

	app, err := initializeApp(log, cfg.DatabaseURL)
	if err != nil {
		return nil, err
	}

	return &AuthApp{
		App:    *app,
		Config: cfg,
	}, nil
}

func NewChatApp() (*ChatApp, error) {
	log, err := initializeLogger("chat")
	if err != nil {
		return nil, fmt.Errorf("failed to initialize logger: %w", err)
	}

	cfg, err := config.LoadChatConfig()
	if err != nil {
		log.Fatalf("failed to load config: %v", err)
		return nil, err
	}

	app, err := initializeApp(log, cfg.DatabaseURL)
	if err != nil {
		return nil, err
	}

	return &ChatApp{
		App:    *app,
		Config: cfg,
	}, nil
}

func initializeApp(log *logger.Logger, databaseURL string) (*App, error) {
	pool := db.NewPool(log, databaseURL)
	if pool == nil {
		return nil, fmt.Errorf("failed to initialize database pool")
	}

	db.StartPoolMetrics(pool, constants.DBPoolMetricsInterval)

	userRepo := userrepo.NewPgRepository(pool)
	identityRepo := identityrepo.NewPgRepository(pool)
	identityService := identityservice.NewIdentityService(identityRepo, log)

	return &App{
		Log:             log,
		Pool:            pool,
		UserRepo:        userRepo,
		IdentityRepo:    identityRepo,
		IdentityService: identityService,
	}, nil
}

func initializeLogger(serviceName string) (*logger.Logger, error) {
	return logger.New(os.Getenv("LOG_DIR"), serviceName, os.Getenv("LOG_LEVEL"))
}
