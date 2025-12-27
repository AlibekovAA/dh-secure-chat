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
	"github.com/AlibekovAA/dh-secure-chat/backend/internal/common/config"
	commoncrypto "github.com/AlibekovAA/dh-secure-chat/backend/internal/common/crypto"
	"github.com/AlibekovAA/dh-secure-chat/backend/internal/common/db"
	commonhttp "github.com/AlibekovAA/dh-secure-chat/backend/internal/common/http"
	"github.com/AlibekovAA/dh-secure-chat/backend/internal/common/logger"
	srv "github.com/AlibekovAA/dh-secure-chat/backend/internal/common/server"
	identityrepo "github.com/AlibekovAA/dh-secure-chat/backend/internal/identity/repository"
	identityservice "github.com/AlibekovAA/dh-secure-chat/backend/internal/identity/service"
	userrepo "github.com/AlibekovAA/dh-secure-chat/backend/internal/user/repository"
)

func main() {
	log, err := logger.New(os.Getenv("LOG_DIR"), "auth", os.Getenv("LOG_LEVEL"))
	if err != nil {
		os.Stderr.WriteString(fmt.Sprintf("failed to initialize logger: %v\n", err))
		os.Exit(1)
	}

	cfg, err := config.LoadAuthConfig()
	if err != nil {
		log.Fatalf("failed to load config: %v", err)
	}

	pool := db.NewPool(log, cfg.DatabaseURL)
	defer pool.Close()

	userRepo := userrepo.NewPgRepository(pool)
	identityRepo := identityrepo.NewPgRepository(pool)
	identityService := identityservice.NewIdentityService(identityRepo, log)
	refreshTokenRepo := authrepo.NewPgRefreshTokenRepository(pool)
	revokedTokenRepo := authrepo.NewPgRevokedTokenRepository(pool)
	hasher := &commoncrypto.BcryptHasher{}
	idGenerator := &commoncrypto.UUIDGenerator{}
	authService := service.NewAuthService(
		userRepo,
		identityService,
		refreshTokenRepo,
		revokedTokenRepo,
		hasher,
		idGenerator,
		cfg.JWTSecret,
		cfg.AccessTokenTTL,
		cfg.RefreshTokenTTL,
		cfg.MaxRefreshTokensPerUser,
		cfg.CircuitBreakerThreshold,
		cfg.CircuitBreakerTimeout,
		cfg.CircuitBreakerReset,
		log,
	)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go authcleanup.StartRefreshTokenCleanup(ctx, refreshTokenRepo, log)
	go authcleanup.StartRevokedTokenCleanup(ctx, revokedTokenRepo, log)

	handler := authhttp.NewHandler(authService, cfg, log)

	mux := http.NewServeMux()
	mux.Handle("/", handler)
	mux.Handle("/metrics", promhttp.Handler())

	rateLimiter := commonhttp.NewStrictRateLimiter()
	baseHandler := commonhttp.BuildBaseHandler("auth", log, mux)

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

	serverConfig := srv.DefaultServerConfig(cfg.HTTPPort)
	server := srv.NewServer(serverConfig, finalHandler)

	shutdownHooks := []srv.ShutdownHook{
		func(ctx context.Context) error {
			log.Infof("auth service: stopping cleanup goroutines")
			cancel()
			return nil
		},
	}

	srv.StartWithGracefulShutdownAndHooks(server, log, "auth", shutdownHooks)
}
