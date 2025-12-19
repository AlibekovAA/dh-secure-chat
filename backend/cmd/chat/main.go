package main

import (
	"context"
	"expvar"
	"net/http"
	"os"
	"sync"
	"time"

	authrepo "github.com/AlibekovAA/dh-secure-chat/backend/internal/auth/repository"
	chathttp "github.com/AlibekovAA/dh-secure-chat/backend/internal/chat/http"
	chatservice "github.com/AlibekovAA/dh-secure-chat/backend/internal/chat/service"
	"github.com/AlibekovAA/dh-secure-chat/backend/internal/chat/websocket"
	"github.com/AlibekovAA/dh-secure-chat/backend/internal/common/config"
	"github.com/AlibekovAA/dh-secure-chat/backend/internal/common/db"
	commonhttp "github.com/AlibekovAA/dh-secure-chat/backend/internal/common/http"
	"github.com/AlibekovAA/dh-secure-chat/backend/internal/common/httpmetrics"
	"github.com/AlibekovAA/dh-secure-chat/backend/internal/common/jwtverify"
	"github.com/AlibekovAA/dh-secure-chat/backend/internal/common/logger"
	srv "github.com/AlibekovAA/dh-secure-chat/backend/internal/common/server"
	identityhttp "github.com/AlibekovAA/dh-secure-chat/backend/internal/identity/http"
	identityrepo "github.com/AlibekovAA/dh-secure-chat/backend/internal/identity/repository"
	identityservice "github.com/AlibekovAA/dh-secure-chat/backend/internal/identity/service"
	userrepo "github.com/AlibekovAA/dh-secure-chat/backend/internal/user/repository"
)

func main() {
	cfg, err := config.LoadChatConfig()
	if err != nil {
		log := logger.GetInstance()
		if initErr := log.Initialize(os.Getenv("LOG_DIR"), "chat", os.Getenv("LOG_LEVEL")); initErr != nil {
			os.Exit(1)
		}
		log.Fatalf("failed to load config: %v", err)
	}

	log := logger.GetInstance()
	if err := log.Initialize(os.Getenv("LOG_DIR"), "chat", os.Getenv("LOG_LEVEL")); err != nil {
		log.Fatalf("failed to initialize logger: %v", err)
	}

	pool := db.NewPool(log, cfg.DatabaseURL)
	defer pool.Close()

	userRepo := userrepo.NewPgRepository(pool)
	identityRepo := identityrepo.NewPgRepository(pool)
	identityService := identityservice.NewIdentityService(identityRepo, log)
	chatSvc := chatservice.NewChatService(userRepo, identityService, log)

	hub := websocket.NewHub(log, userRepo, cfg.LastSeenUpdateInterval)
	ctx, cancel := context.WithCancel(context.Background())
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		hub.Run(ctx)
	}()

	handler := chathttp.NewHandler(chatSvc, hub, cfg, log, pool)

	restMux := http.NewServeMux()
	restMux.HandleFunc("/health", commonhttp.HealthHandler(log))
	restMux.Handle("/debug/vars", expvar.Handler())

	identityHandler := identityhttp.NewHandler(identityService, log)

	revokedTokenRepo := authrepo.NewPgRevokedTokenRepository(pool)
	jwtMw := jwtverify.Middleware(cfg.JWTSecret, log, revokedTokenRepo)
	restMux.Handle("/api/chat/me", jwtMw(handler))
	restMux.Handle("/api/chat/users", jwtMw(handler))
	restMux.Handle("/api/chat/users/", jwtMw(handler))
	restMux.Handle("/api/identity/", jwtMw(identityHandler))

	metrics := httpmetrics.New("chat")
	recovery := commonhttp.RecoveryMiddleware(log)
	traceID := commonhttp.TraceIDMiddleware
	maxRequestSize := commonhttp.MaxRequestSizeMiddleware(commonhttp.DefaultMaxRequestSize)
	wrappedRestMux := recovery(traceID(maxRequestSize(metrics.Wrap(restMux))))

	mainMux := http.NewServeMux()
	mainMux.Handle("/ws/", handler)
	mainMux.Handle("/", wrappedRestMux)

	server := &http.Server{
		Addr:              ":" + cfg.HTTPPort,
		Handler:           mainMux,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       10 * time.Second,
		WriteTimeout:      10 * time.Second,
		IdleTimeout:       120 * time.Second,
	}

	srv.StartWithGracefulShutdown(server, log, "chat")

	cancel()
	wg.Wait()
}
