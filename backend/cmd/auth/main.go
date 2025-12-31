package main

import (
	"context"
	"fmt"
	"net/http"
	"os"

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
	defer app.Pool.Close()

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

	go authcleanup.StartRefreshTokenCleanup(ctx, refreshTokenRepo, app.Log)
	go authcleanup.StartRevokedTokenCleanup(ctx, revokedTokenRepo, app.Log)

	handler := authhttp.NewHandler(authService, app.Config, app.Log)

	mux := http.NewServeMux()
	mux.Handle("/", handler)
	mux.Handle("/metrics", promhttp.Handler())

	baseHandler := commonhttp.BuildBaseHandler("auth", app.Log, mux)
	finalHandler := baseHandler

	serverConfig := srv.DefaultServerConfig(app.Config.HTTPPort)
	server := srv.NewServer(serverConfig, finalHandler)

	shutdownHooks := []srv.ShutdownHook{
		func(ctx context.Context) error {
			app.Log.Infof("auth service: stopping cleanup goroutines")
			cancel()
			return nil
		},
		func(ctx context.Context) error {
			app.Log.Infof("auth service: closing refresh token cache")
			authService.CloseRefreshTokenCache()
			return nil
		},
	}

	srv.StartWithGracefulShutdownAndHooks(server, app.Log, "auth", shutdownHooks)
}
