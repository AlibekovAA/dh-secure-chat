package server

import (
	"context"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/AlibekovAA/dh-secure-chat/backend/internal/common/constants"
	"github.com/AlibekovAA/dh-secure-chat/backend/internal/common/logger"
)

type ShutdownHook func(ctx context.Context) error

func StartWithGracefulShutdown(
	server *http.Server,
	log *logger.Logger,
	serviceName string,
) {
	StartWithGracefulShutdownAndHooks(server, log, serviceName, nil)
}

func StartWithGracefulShutdownAndHooks(
	server *http.Server,
	log *logger.Logger,
	serviceName string,
	hooks []ShutdownHook,
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

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), constants.ShutdownTimeout)
	defer shutdownCancel()

	drainCtx, drainCancel := context.WithTimeout(shutdownCtx, constants.DrainTimeout)
	defer drainCancel()

	log.Infof("%s service: stopping accepting new connections (drain period: %v)", serviceName, constants.DrainTimeout)
	server.SetKeepAlivesEnabled(false)

	if len(hooks) > 0 {
		log.Infof("%s service: executing shutdown hooks", serviceName)
		for i, hook := range hooks {
			if err := hook(drainCtx); err != nil {
				log.Errorf("%s service: shutdown hook %d failed: %v", serviceName, i, err)
			}
		}
	}

	if err := server.Shutdown(shutdownCtx); err != nil {
		log.Errorf("%s service forced to shutdown: %v", serviceName, err)
	} else {
		log.Infof("%s service stopped gracefully", serviceName)
	}
}
