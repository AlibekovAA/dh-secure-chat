package main

import (
	"context"
	"expvar"
	"net/http"
	"os"
	"time"

	authhttp "github.com/AlibekovAA/dh-secure-chat/backend/internal/auth/http"
	authrepo "github.com/AlibekovAA/dh-secure-chat/backend/internal/auth/repository"
	"github.com/AlibekovAA/dh-secure-chat/backend/internal/auth/service"
	"github.com/AlibekovAA/dh-secure-chat/backend/internal/common/config"
	commoncrypto "github.com/AlibekovAA/dh-secure-chat/backend/internal/common/crypto"
	"github.com/AlibekovAA/dh-secure-chat/backend/internal/common/db"
	commonhttp "github.com/AlibekovAA/dh-secure-chat/backend/internal/common/http"
	"github.com/AlibekovAA/dh-secure-chat/backend/internal/common/httpmetrics"
	"github.com/AlibekovAA/dh-secure-chat/backend/internal/common/logger"
	srv "github.com/AlibekovAA/dh-secure-chat/backend/internal/common/server"
	identityrepo "github.com/AlibekovAA/dh-secure-chat/backend/internal/identity/repository"
	identityservice "github.com/AlibekovAA/dh-secure-chat/backend/internal/identity/service"
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
	identityService := identityservice.NewIdentityService(identityRepo, log)
	refreshTokenRepo := authrepo.NewPgRefreshTokenRepository(pool)
	hasher := &commoncrypto.BcryptHasher{}
	idGenerator := &commoncrypto.UUIDGenerator{}
	authService := service.NewAuthService(userRepo, identityService, refreshTokenRepo, hasher, idGenerator, cfg.JWTSecret, log)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go startRefreshTokenCleanup(ctx, refreshTokenRepo, log)

	handler := authhttp.NewHandler(authService, log)

	mux := http.NewServeMux()
	mux.Handle("/", handler)
	mux.Handle("/debug/vars", expvar.Handler())

	metrics := httpmetrics.New("auth")
	recovery := commonhttp.RecoveryMiddleware(log)
	traceID := commonhttp.TraceIDMiddleware

	server := &http.Server{
		Addr:              ":" + cfg.HTTPPort,
		Handler:           recovery(traceID(metrics.Wrap(mux))),
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       10 * time.Second,
		WriteTimeout:      10 * time.Second,
		IdleTimeout:       120 * time.Second,
	}

	srv.StartWithGracefulShutdown(server, log, "auth")

	cancel()
}

func startRefreshTokenCleanup(ctx context.Context, repo authrepo.RefreshTokenRepository, log *logger.Logger) {
	ticker := time.NewTicker(time.Hour)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			deleted, err := repo.DeleteExpired(ctx)
			if err != nil {
				log.Errorf("refresh token cleanup failed: %v", err)
				continue
			}
			if deleted > 0 {
				log.Infof("refresh token cleanup: deleted %d expired tokens", deleted)
			}
		}
	}
}
