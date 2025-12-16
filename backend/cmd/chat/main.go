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
	"github.com/AlibekovAA/dh-secure-chat/backend/internal/common/httpmetrics"
	"github.com/AlibekovAA/dh-secure-chat/backend/internal/common/jwtverify"
	"github.com/AlibekovAA/dh-secure-chat/backend/internal/common/logger"
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

	repo := userrepo.NewPgRepository(pool)
	chatSvc := chatservice.NewChatService(repo, log)
	handler := chathttp.NewHandler(chatSvc, log)

	mux := http.NewServeMux()
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})
	mux.Handle("/debug/vars", expvar.Handler())

	jwtMw := jwtverify.Middleware(cfg.JWTSecret, log)
	mux.Handle("/api/chat/me", jwtMw(handler))
	mux.Handle("/api/chat/users", jwtMw(handler))

	metrics := httpmetrics.New("chat")

	server := &http.Server{
		Addr:              ":" + cfg.HTTPPort,
		Handler:           metrics.Wrap(mux),
		ReadHeaderTimeout: 5 * time.Second,
	}

	log.Infof("chat service listening on :%s", cfg.HTTPPort)
	if err := server.ListenAndServe(); err != nil {
		log.Fatalf("failed to start chat service: %v", err)
	}
}
