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
	"github.com/AlibekovAA/dh-secure-chat/backend/internal/common/httpmetrics"
	"github.com/AlibekovAA/dh-secure-chat/backend/internal/common/logger"
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

	repo := userrepo.NewPgRepository(pool)
	authService := service.NewAuthService(repo, cfg.JWTSecret, log)

	handler := authhttp.NewHandler(authService)

	mux := http.NewServeMux()
	mux.Handle("/", handler)
	mux.Handle("/debug/vars", expvar.Handler())

	metrics := httpmetrics.New("auth")

	server := &http.Server{
		Addr:              ":" + cfg.HTTPPort,
		Handler:           metrics.Wrap(mux),
		ReadHeaderTimeout: 5 * time.Second,
	}

	log.Infof("auth service listening on :%s", cfg.HTTPPort)
	if err := server.ListenAndServe(); err != nil {
		log.Fatalf("failed to start auth service: %v", err)
	}
}
