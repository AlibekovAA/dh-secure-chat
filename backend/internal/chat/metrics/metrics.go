package metrics

import (
	prommetrics "github.com/AlibekovAA/dh-secure-chat/backend/internal/common/prometheus"
)

func IncrementActiveWebSocketConnections() {
	prommetrics.ChatWebSocketConnectionsActive.Inc()
	prommetrics.ChatWebSocketConnectionsTotal.Inc()
}

func DecrementActiveWebSocketConnections() {
	prommetrics.ChatWebSocketConnectionsActive.Dec()
}

func IncrementWebSocketError(errorType string) {
	prommetrics.ChatWebSocketErrors.WithLabelValues(errorType).Inc()
}

func IncrementWebSocketMessage(messageType string) {
	prommetrics.ChatWebSocketMessagesTotal.WithLabelValues(messageType).Inc()
}

func IncrementWebSocketFile() {
	prommetrics.ChatWebSocketFilesTotal.Inc()
}

func IncrementWebSocketFileChunk() {
	prommetrics.ChatWebSocketFilesChunksTotal.Inc()
}
