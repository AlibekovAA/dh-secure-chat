package middleware

import (
	"context"

	"github.com/AlibekovAA/dh-secure-chat/backend/internal/common/logger"
)

type IdempotencyTracker interface {
	GenerateOperationID(userID string, msgType string, payload []byte) string
	Execute(operationID string, msgType string, fn func() (interface{}, error)) (interface{}, error)
}

type Client interface {
	UserID() string
}

type WSMessage struct {
	Type    string
	Payload []byte
}

type IdempotencyMiddleware struct {
	tracker IdempotencyTracker
	log     *logger.Logger
}

func NewIdempotencyMiddleware(tracker IdempotencyTracker, log *logger.Logger) *IdempotencyMiddleware {
	return &IdempotencyMiddleware{
		tracker: tracker,
		log:     log,
	}
}

type MessageHandler func(ctx context.Context, client Client, msg *WSMessage) error

func (m *IdempotencyMiddleware) Handle(ctx context.Context, client Client, msg *WSMessage, handler MessageHandler) error {
	operationID := m.tracker.GenerateOperationID(client.UserID(), msg.Type, msg.Payload)
	return m.HandleWithOperationID(ctx, client, msg, operationID, handler)
}

func (m *IdempotencyMiddleware) HandleWithOperationID(ctx context.Context, client Client, msg *WSMessage, operationID string, handler MessageHandler) error {
	_, err := m.tracker.Execute(operationID, msg.Type, func() (interface{}, error) {
		return nil, handler(ctx, client, msg)
	})

	if err != nil {
		m.log.WithFields(ctx, logger.Fields{
			"user_id": client.UserID(),
			"type":    msg.Type,
			"action":  "idempotency_failed",
		}).Warnf("idempotency check failed: %v", err)
		return err
	}

	return nil
}
