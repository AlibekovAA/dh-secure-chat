package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus/promhttp"

	authrepo "github.com/AlibekovAA/dh-secure-chat/backend/internal/auth/repository"
	chathttp "github.com/AlibekovAA/dh-secure-chat/backend/internal/chat/http"
	chatservice "github.com/AlibekovAA/dh-secure-chat/backend/internal/chat/service"
	"github.com/AlibekovAA/dh-secure-chat/backend/internal/chat/websocket"
	"github.com/AlibekovAA/dh-secure-chat/backend/internal/common/config"
	"github.com/AlibekovAA/dh-secure-chat/backend/internal/common/db"
	commonhttp "github.com/AlibekovAA/dh-secure-chat/backend/internal/common/http"
	"github.com/AlibekovAA/dh-secure-chat/backend/internal/common/jwtverify"
	"github.com/AlibekovAA/dh-secure-chat/backend/internal/common/logger"
	srv "github.com/AlibekovAA/dh-secure-chat/backend/internal/common/server"
	identityhttp "github.com/AlibekovAA/dh-secure-chat/backend/internal/identity/http"
	identityrepo "github.com/AlibekovAA/dh-secure-chat/backend/internal/identity/repository"
	identityservice "github.com/AlibekovAA/dh-secure-chat/backend/internal/identity/service"
	userrepo "github.com/AlibekovAA/dh-secure-chat/backend/internal/user/repository"
)

func main() {
	log, err := logger.New(os.Getenv("LOG_DIR"), "chat", os.Getenv("LOG_LEVEL"))
	if err != nil {
		os.Stderr.WriteString(fmt.Sprintf("failed to initialize logger: %v\n", err))
		os.Exit(1)
	}

	cfg, err := config.LoadChatConfig()
	if err != nil {
		log.Fatalf("failed to load config: %v", err)
	}

	pool := db.NewPool(log, cfg.DatabaseURL)
	defer pool.Close()

	userRepo := userrepo.NewPgRepository(pool)
	identityRepo := identityrepo.NewPgRepository(pool)
	identityService := identityservice.NewIdentityService(identityRepo, log)
	chatSvc := chatservice.NewChatService(userRepo, identityService, log)

	shardCount := 0
	if v := os.Getenv("CHAT_WS_SHARD_COUNT"); v != "" {
		if count, err := strconv.Atoi(v); err == nil && count > 0 {
			shardCount = count
		}
	}

	hubConfig := websocket.HubConfig{
		MaxFileSize:             50 * 1024 * 1024,
		MaxVoiceSize:            10 * 1024 * 1024,
		ProcessorWorkers:        10,
		ProcessorQueueSize:      10000,
		LastSeenUpdateInterval:  cfg.LastSeenUpdateInterval,
		CircuitBreakerThreshold: cfg.CircuitBreakerThreshold,
		CircuitBreakerTimeout:   cfg.CircuitBreakerTimeout,
		CircuitBreakerReset:     cfg.CircuitBreakerReset,
		FileTransferTimeout:     10 * time.Minute,
		IdempotencyTTL:          5 * time.Minute,
		SendTimeout:             cfg.WebSocketSendTimeout,
		ShardCount:              shardCount,
		MaxConnections:          cfg.WebSocketMaxConnections,
		DebugSampleRate:         0.01,
	}

	var hub websocket.HubInterface
	ctx, cancel := context.WithCancel(context.Background())
	var wg sync.WaitGroup

	if shardCount > 0 {
		shardedHub := websocket.NewShardedHub(log, userRepo, hubConfig, shardCount)
		hub = shardedHub
		log.Infof("using sharded hub with %d shards", shardCount)
		wg.Add(1)
		go func() {
			defer wg.Done()
			shardedHub.Run(ctx)
		}()
	} else {
		regularHub := websocket.NewHub(log, userRepo, hubConfig)
		hub = regularHub
		log.Infof("using regular hub")
		wg.Add(1)
		go func() {
			defer wg.Done()
			regularHub.Run(ctx)
		}()
	}

	handler := chathttp.NewHandler(chatSvc, hub, cfg, log, pool)

	restMux := http.NewServeMux()
	restMux.HandleFunc("/health", commonhttp.HealthHandler(log))
	restMux.Handle("/metrics", promhttp.Handler())

	identityHandler := identityhttp.NewHandler(identityService, log)

	revokedTokenRepo := authrepo.NewPgRevokedTokenRepository(pool)
	jwtMw := jwtverify.Middleware(cfg.JWTSecret, log, revokedTokenRepo)
	restMux.Handle("/api/chat/me", jwtMw(handler))
	restMux.Handle("/api/chat/users", jwtMw(handler))
	restMux.Handle("/api/chat/users/", jwtMw(handler))
	restMux.Handle("/api/identity/", jwtMw(identityHandler))

	wrappedRestMux := commonhttp.BuildBaseHandler("chat", log, restMux)

	mainMux := http.NewServeMux()
	mainMux.Handle("/ws/", handler)
	mainMux.Handle("/", wrappedRestMux)

	serverConfig := srv.DefaultServerConfig(cfg.HTTPPort)
	server := srv.NewServer(serverConfig, mainMux)

	shutdownHooks := []srv.ShutdownHook{
		func(ctx context.Context) error {
			log.Infof("chat service: shutting down WebSocket hub")
			hub.Shutdown()
			cancel()
			wg.Wait()
			return nil
		},
	}

	srv.StartWithGracefulShutdownAndHooks(server, log, "chat", shutdownHooks)
}
