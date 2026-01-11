package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"sync"

	"github.com/prometheus/client_golang/prometheus/promhttp"

	authcleanup "github.com/AlibekovAA/dh-secure-chat/backend/internal/auth/cleanup"
	authhttp "github.com/AlibekovAA/dh-secure-chat/backend/internal/auth/http"
	authrepo "github.com/AlibekovAA/dh-secure-chat/backend/internal/auth/repository"
	"github.com/AlibekovAA/dh-secure-chat/backend/internal/auth/service"
	"github.com/AlibekovAA/dh-secure-chat/backend/internal/common/bootstrap"
	commoncrypto "github.com/AlibekovAA/dh-secure-chat/backend/internal/common/crypto"
	commonhttp "github.com/AlibekovAA/dh-secure-chat/backend/internal/common/http"
	srv "github.com/AlibekovAA/dh-secure-chat/backend/internal/common/server"
)

func main() {
	app, err := bootstrap.NewAuthApp()
	if err != nil {
		os.Stderr.WriteString(fmt.Sprintf("failed to initialize app: %v\n", err))
		os.Exit(1)
	}

	refreshTokenRepo := authrepo.NewPgRefreshTokenRepository(app.Pool)
	revokedTokenRepo := authrepo.NewPgRevokedTokenRepository(app.Pool)
	hasher := &commoncrypto.BcryptHasher{}
	idGenerator := &commoncrypto.UUIDGenerator{}
	authService := service.NewAuthService(
		service.AuthServiceDeps{
			Repo:             app.UserRepo,
			IdentityService:  app.IdentityService,
			RefreshTokenRepo: refreshTokenRepo,
			RevokedTokenRepo: revokedTokenRepo,
			Hasher:           hasher,
			IDGenerator:      idGenerator,
			Log:              app.Log,
		},
		service.AuthServiceConfig{
			JWTSecret:               app.Config.JWTSecret,
			AccessTokenTTL:          app.Config.AccessTokenTTL,
			RefreshTokenTTL:         app.Config.RefreshTokenTTL,
			MaxRefreshTokens:        app.Config.MaxRefreshTokensPerUser,
			CircuitBreakerThreshold: app.Config.CircuitBreakerThreshold,
			CircuitBreakerTimeout:   app.Config.CircuitBreakerTimeout,
			CircuitBreakerReset:     app.Config.CircuitBreakerReset,
		},
	)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var cleanupWg sync.WaitGroup
	cleanupWg.Add(2)
	go func() {
		defer cleanupWg.Done()
		authcleanup.StartRefreshTokenCleanup(ctx, refreshTokenRepo, app.Log)
	}()
	go func() {
		defer cleanupWg.Done()
		authcleanup.StartRevokedTokenCleanup(ctx, revokedTokenRepo, app.Log)
	}()

	handler := authhttp.NewHandler(authService, app.Config, app.Log)

	mux := http.NewServeMux()
	mux.HandleFunc("/health", commonhttp.HealthHandler(app.Log))
	mux.Handle("/metrics", promhttp.Handler())
	mux.Handle("/", handler)

	baseHandler := commonhttp.BuildBaseHandler("auth", app.Log, mux)
	finalHandler := baseHandler

	serverConfig := srv.DefaultServerConfig(app.Config.HTTPPort)
	server := srv.NewServer(serverConfig, finalHandler)

	shutdownHooks := []srv.ShutdownHook{
		func(ctx context.Context) error {
			app.Log.Infof("auth service: stopping cleanup goroutines")
			cancel()
			srv.WaitGroupWithTimeout(ctx, &cleanupWg, app.Log, "auth service: cleanup goroutines stopped")
			return nil
		},
		func(ctx context.Context) error {
			app.Log.Infof("auth service: closing refresh token cache")
			authService.CloseRefreshTokenCache()
			return nil
		},
		func(ctx context.Context) error {
			app.Log.Infof("auth service: closing database pool")
			srv.ClosePoolWithTimeout(ctx, app.Pool, app.Log, "auth")
			return nil
		},
	}

	srv.StartWithGracefulShutdownAndHooks(server, app.Log, "auth", shutdownHooks)
}
