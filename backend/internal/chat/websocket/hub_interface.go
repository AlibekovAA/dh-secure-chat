package websocket

import "context"

type HubInterface interface {
	Register(client *Client)
	Unregister(client *Client)
	Run(ctx context.Context)
	SendToUser(userID string, message *WSMessage) bool
	SendToUserWithContext(ctx context.Context, userID string, message *WSMessage) error
	IsUserOnline(userID string) bool
	HandleMessage(client *Client, msg *WSMessage)
	Shutdown()
}
