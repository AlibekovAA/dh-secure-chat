package websocket

import "context"

type MessageSender interface {
	SendToUserWithContext(ctx context.Context, userID string, message *WSMessage) error
	SendErrorToUser(userID string, err error)
	IsUserOnline(userID string) bool
}
