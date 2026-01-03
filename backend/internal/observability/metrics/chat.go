package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	ChatWebSocketConnectionsActive = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "chat_websocket_connections_active",
			Help: "Number of active WebSocket connections",
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

	ChatWebSocketDroppedMessages = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "chat_websocket_dropped_messages_total",
			Help: "Total number of dropped messages due to slow clients",
		},
		[]string{"message_type"},
	)

	ChatWebSocketConnectionsRejected = promauto.NewCounter(
		prometheus.CounterOpts{
			Name: "chat_websocket_connections_rejected_total",
			Help: "Total number of WebSocket connections rejected due to max connections limit",
		},
	)

	ChatWebSocketUserExistenceCacheHits = promauto.NewCounter(
		prometheus.CounterOpts{
			Name: "chat_websocket_user_existence_cache_hits_total",
			Help: "Total number of user existence cache hits",
		},
	)

	ChatWebSocketUserExistenceCacheMisses = promauto.NewCounter(
		prometheus.CounterOpts{
			Name: "chat_websocket_user_existence_cache_misses_total",
			Help: "Total number of user existence cache misses",
		},
	)

	ChatWebSocketUserExistenceCacheSize = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "chat_websocket_user_existence_cache_size",
			Help: "Current size of user existence cache",
		},
	)

	ChatWebSocketMessageSendDurationSeconds = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "chat_websocket_message_send_duration_seconds",
			Help:    "Duration of WebSocket message send operations in seconds",
			Buckets: []float64{0.001, 0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1},
		},
		[]string{"message_type"},
	)
)
