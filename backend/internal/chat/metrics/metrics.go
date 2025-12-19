package metrics

import "expvar"

var (
	ActiveWebSocketConnections = expvar.NewInt("chat_websocket_connections_active")
	WebSocketConnectionsTotal  = expvar.NewInt("chat_websocket_connections_total")
	WebSocketErrors            = expvar.NewMap("chat_websocket_errors")
	WebSocketMessagesTotal     = expvar.NewMap("chat_websocket_messages_total")
	WebSocketFilesTotal        = expvar.NewInt("chat_websocket_files_total")
	WebSocketFilesChunksTotal  = expvar.NewInt("chat_websocket_files_chunks_total")
)

func IncrementActiveWebSocketConnections() {
	ActiveWebSocketConnections.Add(1)
	WebSocketConnectionsTotal.Add(1)
}

func DecrementActiveWebSocketConnections() {
	ActiveWebSocketConnections.Add(-1)
}

func IncrementWebSocketError(errorType string) {
	WebSocketErrors.Add(errorType, 1)
}

func IncrementWebSocketMessage(messageType string) {
	WebSocketMessagesTotal.Add(messageType, 1)
}

func IncrementWebSocketFile() {
	WebSocketFilesTotal.Add(1)
}

func IncrementWebSocketFileChunk() {
	WebSocketFilesChunksTotal.Add(1)
}
