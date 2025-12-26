package metrics

import (
	observabilitymetrics "github.com/AlibekovAA/dh-secure-chat/backend/internal/observability/metrics"
)

func IncrementActiveWebSocketConnections() {
	observabilitymetrics.ChatWebSocketConnectionsActive.Inc()
	observabilitymetrics.ChatWebSocketConnectionsTotal.Inc()
}

func DecrementActiveWebSocketConnections() {
	observabilitymetrics.ChatWebSocketConnectionsActive.Dec()
}

func IncrementWebSocketError(errorType string) {
	observabilitymetrics.ChatWebSocketErrors.WithLabelValues(errorType).Inc()
}

func IncrementWebSocketMessage(messageType string) {
	observabilitymetrics.ChatWebSocketMessagesTotal.WithLabelValues(messageType).Inc()
}

func IncrementWebSocketFile() {
	observabilitymetrics.ChatWebSocketFilesTotal.Inc()
}

func IncrementWebSocketFileChunk() {
	observabilitymetrics.ChatWebSocketFilesChunksTotal.Inc()
}
