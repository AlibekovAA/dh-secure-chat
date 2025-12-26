package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
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
