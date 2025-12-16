package main

import (
	"expvar"
	"net/http"
	"os"
	"time"

	chathttp "github.com/AlibekovAA/dh-secure-chat/backend/internal/chat/http"
	chatservice "github.com/AlibekovAA/dh-secure-chat/backend/internal/chat/service"
	"github.com/AlibekovAA/dh-secure-chat/backend/internal/common/config"
	"github.com/AlibekovAA/dh-secure-chat/backend/internal/common/db"
	commonhttp "github.com/AlibekovAA/dh-secure-chat/backend/internal/common/http"
	"github.com/AlibekovAA/dh-secure-chat/backend/internal/common/httpmetrics"
	"github.com/AlibekovAA/dh-secure-chat/backend/internal/common/jwtverify"
	"github.com/AlibekovAA/dh-secure-chat/backend/internal/common/logger"
	srv "github.com/AlibekovAA/dh-secure-chat/backend/internal/common/server"
	identityrepo "github.com/AlibekovAA/dh-secure-chat/backend/internal/identity/repository"
	userrepo "github.com/AlibekovAA/dh-secure-chat/backend/internal/user/repository"
)

func main() {
	log := logger.GetInstance()
	if err := log.Initialize(os.Getenv("LOG_DIR"), "chat", os.Getenv("LOG_LEVEL")); err != nil {
		log.Fatalf("failed to initialize logger: %v", err)
	}

	cfg := config.LoadChatConfig()

	pool := db.NewPool(log, cfg.DatabaseURL)
	defer pool.Close()

	userRepo := userrepo.NewPgRepository(pool)
	identityRepo := identityrepo.NewPgRepository(pool)
	chatSvc := chatservice.NewChatService(userRepo, identityRepo, log)
	handler := chathttp.NewHandler(chatSvc, log)

	mux := http.NewServeMux()
	mux.HandleFunc("/health", commonhttp.HealthHandler(log))
	mux.Handle("/debug/vars", expvar.Handler())

	jwtMw := jwtverify.Middleware(cfg.JWTSecret, log)
	mux.Handle("/api/chat/me", jwtMw(handler))
	mux.Handle("/api/chat/users", jwtMw(handler))
	mux.Handle("/api/chat/users/", jwtMw(handler))

	metrics := httpmetrics.New("chat")
	recovery := commonhttp.RecoveryMiddleware(log)

	server := &http.Server{
		Addr:              ":" + cfg.HTTPPort,
		Handler:           recovery(metrics.Wrap(mux)),
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       10 * time.Second,
		WriteTimeout:      10 * time.Second,
		IdleTimeout:       120 * time.Second,
	}

	srv.StartWithGracefulShutdown(server, log, "chat")
}
