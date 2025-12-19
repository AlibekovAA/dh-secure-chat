package main

import (
	"context"
	"expvar"
	"net/http"
	"os"
	"time"

	authcleanup "github.com/AlibekovAA/dh-secure-chat/backend/internal/auth/cleanup"
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
	cfg, err := config.LoadAuthConfig()
	if err != nil {
		log := logger.GetInstance()
		if initErr := log.Initialize(os.Getenv("LOG_DIR"), "auth", os.Getenv("LOG_LEVEL")); initErr != nil {
			os.Exit(1)
		}
		log.Fatalf("failed to load config: %v", err)
	}

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
	revokedTokenRepo := authrepo.NewPgRevokedTokenRepository(pool)
	hasher := &commoncrypto.BcryptHasher{}
	idGenerator := &commoncrypto.UUIDGenerator{}
	authService := service.NewAuthService(userRepo, identityService, refreshTokenRepo, revokedTokenRepo, hasher, idGenerator, cfg.JWTSecret, log)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go authcleanup.StartRefreshTokenCleanup(ctx, refreshTokenRepo, log)
	go authcleanup.StartRevokedTokenCleanup(ctx, revokedTokenRepo, log)

	handler := authhttp.NewHandler(authService, cfg, log)

	mux := http.NewServeMux()
	mux.Handle("/", handler)
	mux.Handle("/debug/vars", expvar.Handler())

	rateLimiter := commonhttp.NewStrictRateLimiter()
	metrics := httpmetrics.New("auth")
	recovery := commonhttp.RecoveryMiddleware(log)
	traceID := commonhttp.TraceIDMiddleware
	maxRequestSize := commonhttp.MaxRequestSizeMiddleware(commonhttp.DefaultMaxRequestSize)
	securityHeaders := commonhttp.SecurityHeadersMiddleware
	csp := commonhttp.ContentSecurityPolicyMiddleware("")

	baseHandler := securityHeaders(csp(recovery(traceID(maxRequestSize(metrics.Wrap(mux))))))

	muxWithRateLimit := http.NewServeMux()
	muxWithRateLimit.HandleFunc("/api/auth/login", rateLimiter.MiddlewareForPath("/api/auth/login")(baseHandler).ServeHTTP)
	muxWithRateLimit.HandleFunc("/api/auth/register", rateLimiter.MiddlewareForPath("/api/auth/register")(baseHandler).ServeHTTP)
	muxWithRateLimit.Handle("/", baseHandler)

	server := &http.Server{
		Addr:              ":" + cfg.HTTPPort,
		Handler:           muxWithRateLimit,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       10 * time.Second,
		WriteTimeout:      10 * time.Second,
		IdleTimeout:       120 * time.Second,
	}

	srv.StartWithGracefulShutdown(server, log, "auth")

	cancel()
}
