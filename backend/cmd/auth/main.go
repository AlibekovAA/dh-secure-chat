package main

import (
	"expvar"
	"fmt"
	"net/http"
	"os"
	"time"

	authhttp "github.com/AlibekovAA/dh-secure-chat/backend/internal/auth/http"
	"github.com/AlibekovAA/dh-secure-chat/backend/internal/auth/repository"
	"github.com/AlibekovAA/dh-secure-chat/backend/internal/auth/service"
	"github.com/AlibekovAA/dh-secure-chat/backend/internal/common/config"
	"github.com/AlibekovAA/dh-secure-chat/backend/internal/common/db"
	"github.com/AlibekovAA/dh-secure-chat/backend/internal/common/logger"
)

var (
	totalRequests      = expvar.NewInt("auth_requests_total")
	inFlightRequests   = expvar.NewInt("auth_requests_in_flight")
	requestDurationMs  = expvar.NewMap("auth_request_duration_ms")
	lastRequestStatus  = expvar.NewMap("auth_last_status_by_path")
	lastRequestSeconds = expvar.NewMap("auth_last_duration_seconds_by_path")
)

func main() {
	cfg := config.LoadAuthConfig()

	log := logger.GetInstance()
	if err := log.Initialize(os.Getenv("LOG_DIR"), "auth", os.Getenv("LOG_LEVEL")); err != nil {
		log.Fatalf("failed to initialize logger: %v", err)
	}

	pool := db.NewPool(cfg.DatabaseURL)
	defer pool.Close()

	repo := repository.NewPgUserRepository(pool)
	authService := service.NewAuthService(repo, cfg.JWTSecret)

	handler := authhttp.NewHandler(authService)

	mux := http.NewServeMux()
	mux.Handle("/", handler)
	mux.Handle("/debug/vars", expvar.Handler())

	server := &http.Server{
		Addr:              ":" + cfg.HTTPPort,
		Handler:           metricsMiddleware(mux),
		ReadHeaderTimeout: 5 * time.Second,
	}

	log.Infof("auth service listening on :%s", cfg.HTTPPort)
	if err := server.ListenAndServe(); err != nil {
		log.Fatalf("failed to start auth service: %v", err)
	}
}

type statusRecorder struct {
	http.ResponseWriter
	status int
}

func (r *statusRecorder) WriteHeader(code int) {
	r.status = code
	r.ResponseWriter.WriteHeader(code)
}

func metricsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		inFlightRequests.Add(1)
		totalRequests.Add(1)

		pathKey := r.Method + " " + r.URL.Path

		rec := &statusRecorder{ResponseWriter: w, status: http.StatusOK}
		next.ServeHTTP(rec, r)

		inFlightRequests.Add(-1)

		elapsed := time.Since(start)
		statusClass := fmt.Sprintf("%dxx", rec.status/100)
		requestDurationMs.Add(pathKey, elapsed.Milliseconds())
		lastRequestStatus.Set(pathKey, expvar.Func(func() any { return statusClass }))
		lastRequestSeconds.Set(pathKey, expvar.Func(func() any { return elapsed.Seconds() }))
	})
}
