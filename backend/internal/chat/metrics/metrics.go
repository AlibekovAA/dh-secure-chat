package metrics

import "expvar"

var (
	ActiveWebSocketConnections = expvar.NewInt("chat_websocket_connections_active")
	ActiveChatSessions         = expvar.NewInt("chat_sessions_active")
	WebSocketConnectionsTotal  = expvar.NewInt("chat_websocket_connections_total")
	ChatSessionsTotal          = expvar.NewInt("chat_sessions_total")
	WebSocketErrors            = expvar.NewMap("chat_websocket_errors")
	ChatSessionErrors          = expvar.NewMap("chat_session_errors")
)

func IncrementActiveWebSocketConnections() {
	ActiveWebSocketConnections.Add(1)
	WebSocketConnectionsTotal.Add(1)
}

func DecrementActiveWebSocketConnections() {
	ActiveWebSocketConnections.Add(-1)
}

func IncrementActiveChatSessions() {
	ActiveChatSessions.Add(1)
	ChatSessionsTotal.Add(1)
}

func DecrementActiveChatSessions() {
	ActiveChatSessions.Add(-1)
}

func IncrementWebSocketError(errorType string) {
	WebSocketErrors.Add(errorType, 1)
}

func IncrementChatSessionError(errorType string) {
	ChatSessionErrors.Add(errorType, 1)
}
