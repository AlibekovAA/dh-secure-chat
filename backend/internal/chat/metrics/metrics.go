package metrics

import "expvar"

var (
	ActiveWebSocketConnections = expvar.NewInt("chat_websocket_connections_active")
	WebSocketConnectionsTotal  = expvar.NewInt("chat_websocket_connections_total")
	WebSocketErrors            = expvar.NewMap("chat_websocket_errors")
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
