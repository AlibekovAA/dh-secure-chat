package main

import (
	"expvar"
	"net/http"
	"os"
	"time"

	authhttp "github.com/AlibekovAA/dh-secure-chat/backend/internal/auth/http"
	"github.com/AlibekovAA/dh-secure-chat/backend/internal/auth/service"
	"github.com/AlibekovAA/dh-secure-chat/backend/internal/common/config"
	"github.com/AlibekovAA/dh-secure-chat/backend/internal/common/db"
	commonhttp "github.com/AlibekovAA/dh-secure-chat/backend/internal/common/http"
	"github.com/AlibekovAA/dh-secure-chat/backend/internal/common/httpmetrics"
	"github.com/AlibekovAA/dh-secure-chat/backend/internal/common/logger"
	srv "github.com/AlibekovAA/dh-secure-chat/backend/internal/common/server"
	identityrepo "github.com/AlibekovAA/dh-secure-chat/backend/internal/identity/repository"
	userrepo "github.com/AlibekovAA/dh-secure-chat/backend/internal/user/repository"
)

func main() {
	cfg := config.LoadAuthConfig()

	log := logger.GetInstance()
	if err := log.Initialize(os.Getenv("LOG_DIR"), "auth", os.Getenv("LOG_LEVEL")); err != nil {
		log.Fatalf("failed to initialize logger: %v", err)
	}

	pool := db.NewPool(log, cfg.DatabaseURL)
	defer pool.Close()

	userRepo := userrepo.NewPgRepository(pool)
	identityRepo := identityrepo.NewPgRepository(pool)
	identityAdapter := service.NewIdentityRepoAdapter(identityRepo)
	authService := service.NewAuthService(userRepo, identityAdapter, cfg.JWTSecret, log)

	handler := authhttp.NewHandler(authService, log)

	mux := http.NewServeMux()
	mux.Handle("/", handler)
	mux.Handle("/debug/vars", expvar.Handler())

	metrics := httpmetrics.New("auth")
	recovery := commonhttp.RecoveryMiddleware(log)

	server := &http.Server{
		Addr:              ":" + cfg.HTTPPort,
		Handler:           recovery(metrics.Wrap(mux)),
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       10 * time.Second,
		WriteTimeout:      10 * time.Second,
		IdleTimeout:       120 * time.Second,
	}

	srv.StartWithGracefulShutdown(server, log, "auth")
}
