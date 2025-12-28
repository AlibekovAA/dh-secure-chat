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
		app.UserRepo,
		app.IdentityService,
		refreshTokenRepo,
		revokedTokenRepo,
		hasher,
		idGenerator,
		app.Config.JWTSecret,
		app.Config.AccessTokenTTL,
		app.Config.RefreshTokenTTL,
		app.Config.MaxRefreshTokensPerUser,
		app.Config.CircuitBreakerThreshold,
		app.Config.CircuitBreakerTimeout,
		app.Config.CircuitBreakerReset,
		app.Log,
	)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go authcleanup.StartRefreshTokenCleanup(ctx, refreshTokenRepo, app.Log)
	go authcleanup.StartRevokedTokenCleanup(ctx, revokedTokenRepo, app.Log)

	handler := authhttp.NewHandler(authService, app.Config, app.Log)

	mux := http.NewServeMux()
	mux.Handle("/", handler)
	mux.Handle("/metrics", promhttp.Handler())

	rateLimiter := commonhttp.NewStrictRateLimiter()
	baseHandler := commonhttp.BuildBaseHandler("auth", app.Log, mux)

	rateLimitMiddleware := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			path := r.URL.Path
			if path == "/health" || path == "/metrics" {
				next.ServeHTTP(w, r)
				return
			}
			rateLimiter.MiddlewareForPath(path)(next).ServeHTTP(w, r)
		})
	}

	finalHandler := rateLimitMiddleware(baseHandler)

	serverConfig := srv.DefaultServerConfig(app.Config.HTTPPort)
	server := srv.NewServer(serverConfig, finalHandler)

	shutdownHooks := []srv.ShutdownHook{
		func(ctx context.Context) error {
			app.Log.Infof("auth service: stopping cleanup goroutines")
			cancel()
			return nil
		},
	}

	srv.StartWithGracefulShutdownAndHooks(server, app.Log, "auth", shutdownHooks)
}
