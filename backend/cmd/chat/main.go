package main

import (
	"net/http"
	"os"

	"github.com/AlibekovAA/dh-secure-chat/backend/internal/common/logger"
)

func main() {
	log := logger.GetInstance()
	if err := log.Initialize(os.Getenv("LOG_DIR"), "chat", os.Getenv("LOG_LEVEL")); err != nil {
		log.Fatalf("failed to initialize logger: %v", err)
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})

	server := &http.Server{
		Addr:    ":8082",
		Handler: mux,
	}

	log.Info("chat service listening on :8082")
	if err := server.ListenAndServe(); err != nil {
		log.Fatalf("failed to start chat service: %v", err)
	}
}
