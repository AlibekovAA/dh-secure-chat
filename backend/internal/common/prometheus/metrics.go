package prometheus

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	AuthRequestsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "auth_requests_total",
			Help: "Total number of auth requests",
		},
		[]string{"method", "path"},
	)

	AuthRequestsInFlight = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "auth_requests_in_flight",
			Help: "Number of auth requests currently being processed",
		},
	)

	AuthRequestDurationSeconds = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "auth_request_duration_seconds",
			Help:    "Duration of auth requests in seconds",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"method", "path", "status"},
	)

	ChatRequestsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "chat_requests_total",
			Help: "Total number of chat requests",
		},
		[]string{"method", "path"},
	)

	ChatRequestsInFlight = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "chat_requests_in_flight",
			Help: "Number of chat requests currently being processed",
		},
	)

	ChatRequestDurationSeconds = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "chat_request_duration_seconds",
			Help:    "Duration of chat requests in seconds",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"method", "path", "status"},
	)

	RefreshTokensIssued = promauto.NewCounter(
		prometheus.CounterOpts{
			Name: "refresh_tokens_issued_total",
			Help: "Total number of refresh tokens issued",
		},
	)

	RefreshTokensUsed = promauto.NewCounter(
		prometheus.CounterOpts{
			Name: "refresh_tokens_used_total",
			Help: "Total number of refresh tokens used",
		},
	)

	RefreshTokensRevoked = promauto.NewCounter(
		prometheus.CounterOpts{
			Name: "refresh_tokens_revoked_total",
			Help: "Total number of refresh tokens revoked",
		},
	)

	RefreshTokensExpired = promauto.NewCounter(
		prometheus.CounterOpts{
			Name: "refresh_tokens_expired_total",
			Help: "Total number of expired refresh tokens",
		},
	)

	RefreshTokensCleanupDeleted = promauto.NewCounter(
		prometheus.CounterOpts{
			Name: "refresh_tokens_cleanup_deleted_total",
			Help: "Total number of expired refresh tokens deleted during cleanup",
		},
	)

	AccessTokensIssued = promauto.NewCounter(
		prometheus.CounterOpts{
			Name: "access_tokens_issued_total",
			Help: "Total number of access tokens issued",
		},
	)

	AccessTokensRevoked = promauto.NewCounter(
		prometheus.CounterOpts{
			Name: "access_tokens_revoked_total",
			Help: "Total number of access tokens revoked",
		},
	)

	RevokedTokensCleanupDeleted = promauto.NewCounter(
		prometheus.CounterOpts{
			Name: "revoked_tokens_cleanup_deleted_total",
			Help: "Total number of expired revoked tokens deleted during cleanup",
		},
	)

	JWTValidationsTotal = promauto.NewCounter(
		prometheus.CounterOpts{
			Name: "jwt_validations_total",
			Help: "Total number of JWT validations",
		},
	)

	JWTValidationsFailed = promauto.NewCounter(
		prometheus.CounterOpts{
			Name: "jwt_validations_failed_total",
			Help: "Total number of failed JWT validations",
		},
	)

	JWTRevokedChecksTotal = promauto.NewCounter(
		prometheus.CounterOpts{
			Name: "jwt_revoked_checks_total",
			Help: "Total number of revoked token checks",
		},
	)

	ChatWebSocketConnectionsActive = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "chat_websocket_connections_active",
			Help: "Number of active WebSocket connections",
		},
	)

	ChatWebSocketConnectionsTotal = promauto.NewCounter(
		prometheus.CounterOpts{
			Name: "chat_websocket_connections_total",
			Help: "Total number of WebSocket connections established",
		},
	)

	ChatWebSocketErrors = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "chat_websocket_errors_total",
			Help: "Total number of WebSocket errors by type",
		},
		[]string{"error_type"},
	)

	ChatWebSocketMessagesTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "chat_websocket_messages_total",
			Help: "Total number of WebSocket messages by type",
		},
		[]string{"message_type"},
	)

	ChatWebSocketFilesTotal = promauto.NewCounter(
		prometheus.CounterOpts{
			Name: "chat_websocket_files_total",
			Help: "Total number of files sent via WebSocket",
		},
	)

	ChatWebSocketFilesChunksTotal = promauto.NewCounter(
		prometheus.CounterOpts{
			Name: "chat_websocket_files_chunks_total",
			Help: "Total number of file chunks sent via WebSocket",
		},
	)

	DBPoolAcquiredConnections = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "db_pool_acquired_connections",
			Help: "Number of acquired database connections",
		},
	)

	DBPoolIdleConnections = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "db_pool_idle_connections",
			Help: "Number of idle database connections",
		},
	)

	DBPoolMaxConnections = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "db_pool_max_connections",
			Help: "Maximum number of database connections",
		},
	)

	DBPoolTotalConnections = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "db_pool_total_connections",
			Help: "Total number of database connections",
		},
	)

	DBQueryDurationSeconds = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "db_query_duration_seconds",
			Help:    "Duration of database queries in seconds",
			Buckets: []float64{0.001, 0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1, 2.5, 5},
		},
		[]string{"operation", "table"},
	)

	DBQueryErrors = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "db_query_errors_total",
			Help: "Total number of database query errors",
		},
		[]string{"operation", "table", "error_type"},
	)

	RateLimitBlocked = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "rate_limit_blocked_total",
			Help: "Total number of requests blocked by rate limiter",
		},
		[]string{"path", "limiter_type"},
	)

	CircuitBreakerState = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "circuit_breaker_state",
			Help: "Circuit breaker state (0=closed, 1=open, 2=half-open)",
		},
		[]string{"name"},
	)

	CircuitBreakerFailures = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "circuit_breaker_failures_total",
			Help: "Total number of circuit breaker failures",
		},
		[]string{"name"},
	)

	ChatWebSocketDisconnections = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "chat_websocket_disconnections_total",
			Help: "Total number of WebSocket disconnections",
		},
		[]string{"reason"},
	)

	ChatWebSocketMessageProcessingDurationSeconds = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "chat_websocket_message_processing_duration_seconds",
			Help:    "Duration of WebSocket message processing in seconds",
			Buckets: []float64{0.001, 0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1, 2.5, 5},
		},
		[]string{"message_type"},
	)

	ChatWebSocketMessageProcessorQueueSize = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "chat_websocket_message_processor_queue_size",
			Help: "Current size of WebSocket message processor queue",
		},
	)

	ChatWebSocketFileTransferDurationSeconds = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "chat_websocket_file_transfer_duration_seconds",
			Help:    "Duration of file transfers in seconds",
			Buckets: []float64{1, 5, 10, 30, 60, 120, 300, 600},
		},
		[]string{"status"},
	)

	ChatWebSocketFileTransferFailures = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "chat_websocket_file_transfer_failures_total",
			Help: "Total number of failed file transfers",
		},
		[]string{"reason"},
	)

	ChatWebSocketIdempotencyDuplicates = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "chat_websocket_idempotency_duplicates_total",
			Help: "Total number of duplicate messages detected by idempotency",
		},
		[]string{"message_type"},
	)
)
