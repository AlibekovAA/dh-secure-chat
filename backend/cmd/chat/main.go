package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"sync"

	"github.com/prometheus/client_golang/prometheus/promhttp"

	authrepo "github.com/AlibekovAA/dh-secure-chat/backend/internal/auth/repository"
	chathttp "github.com/AlibekovAA/dh-secure-chat/backend/internal/chat/http"
	chatservice "github.com/AlibekovAA/dh-secure-chat/backend/internal/chat/service"
	"github.com/AlibekovAA/dh-secure-chat/backend/internal/chat/websocket"
	"github.com/AlibekovAA/dh-secure-chat/backend/internal/common/bootstrap"
	"github.com/AlibekovAA/dh-secure-chat/backend/internal/common/clock"
	"github.com/AlibekovAA/dh-secure-chat/backend/internal/common/constants"
	commonhttp "github.com/AlibekovAA/dh-secure-chat/backend/internal/common/http"
	"github.com/AlibekovAA/dh-secure-chat/backend/internal/common/jwtverify"
	srv "github.com/AlibekovAA/dh-secure-chat/backend/internal/common/server"
	identityhttp "github.com/AlibekovAA/dh-secure-chat/backend/internal/identity/http"
)

func main() {
	app, err := bootstrap.NewChatApp()
	if err != nil {
		os.Stderr.WriteString(fmt.Sprintf("failed to initialize app: %v\n", err))
		os.Exit(1)
	}

	chatSvc := chatservice.NewChatService(chatservice.ChatServiceDeps{
		Repo:            app.UserRepo,
		IdentityService: app.IdentityService,
		Log:             app.Log,
	})

	hubConfig := websocket.HubConfig{
		MaxFileSize:             constants.MaxFileSizeBytes,
		MaxVoiceSize:            constants.MaxVoiceSizeBytes,
		ProcessorWorkers:        constants.WebSocketProcessorWorkers,
		ProcessorQueueSize:      constants.WebSocketProcessorQueueSize,
		LastSeenUpdateInterval:  app.Config.LastSeenUpdateInterval,
		CircuitBreakerThreshold: app.Config.CircuitBreakerThreshold,
		CircuitBreakerTimeout:   app.Config.CircuitBreakerTimeout,
		CircuitBreakerReset:     app.Config.CircuitBreakerReset,
		FileTransferTimeout:     constants.FileTransferTimeout,
		IdempotencyTTL:          constants.IdempotencyTTL,
		SendTimeout:             app.Config.WebSocketSendTimeout,
		MaxConnections:          app.Config.WebSocketMaxConnections,
		DebugSampleRate:         constants.WebSocketDebugSampleRate,
	}

	hub := websocket.NewHub(websocket.HubDeps{
		Log:      app.Log,
		UserRepo: app.UserRepo,
		Clock:    clock.NewRealClock(),
	}, hubConfig)
	ctx, cancel := context.WithCancel(context.Background())
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		hub.Run(ctx)
	}()

	handler := chathttp.NewHandler(chatSvc, hub, app.Config, app.Log, app.Pool)

	restMux := http.NewServeMux()
	restMux.HandleFunc("/health", commonhttp.HealthHandler(app.Log))
	restMux.Handle("/metrics", promhttp.Handler())

	identityHandler := identityhttp.NewHandler(app.IdentityService, app.Log)

	revokedTokenRepo := authrepo.NewPgRevokedTokenRepository(app.Pool)
	jwtMw := jwtverify.Middleware(app.Config.JWTSecret, app.Log, revokedTokenRepo)
	restMux.Handle("/api/chat/me", jwtMw(handler))
	restMux.Handle("/api/chat/users", jwtMw(handler))
	restMux.Handle("/api/chat/users/", jwtMw(handler))
	restMux.Handle("/api/identity/", jwtMw(identityHandler))

	wrappedRestMux := commonhttp.BuildBaseHandler("chat", app.Log, restMux)

	mainMux := http.NewServeMux()
	mainMux.Handle("/ws/", handler)
	mainMux.Handle("/", wrappedRestMux)

	serverConfig := srv.DefaultServerConfig(app.Config.HTTPPort)
	server := srv.NewServer(serverConfig, mainMux)

	shutdownHooks := []srv.ShutdownHook{
		func(ctx context.Context) error {
			app.Log.Infof("chat service: shutting down WebSocket hub")
			hub.Shutdown()
			cancel()
			srv.WaitGroupWithTimeout(ctx, &wg, app.Log, "chat service: WebSocket hub stopped")
			return nil
		},
		func(ctx context.Context) error {
			app.Log.Infof("chat service: closing database pool")
			srv.ClosePoolWithTimeout(ctx, app.Pool, app.Log, "chat")
			return nil
		},
	}

	srv.StartWithGracefulShutdownAndHooks(server, app.Log, "chat", shutdownHooks)
}
