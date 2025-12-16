package server

import (
	"context"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/AlibekovAA/dh-secure-chat/backend/internal/common/logger"
)

func StartWithGracefulShutdown(
	server *http.Server,
	log *logger.Logger,
	serviceName string,
) {
	go func() {
		log.Infof("%s service listening on %s", serviceName, server.Addr)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("failed to start %s service: %v", serviceName, err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Infof("shutting down %s service...", serviceName)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		log.Errorf("%s service forced to shutdown: %v", serviceName, err)
	} else {
		log.Infof("%s service stopped gracefully", serviceName)
	}
}
