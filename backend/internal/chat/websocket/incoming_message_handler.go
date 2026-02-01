package websocket

import (
	"context"
	"encoding/json"
	"strconv"

	"github.com/AlibekovAA/dh-secure-chat/backend/internal/chat/websocket/middleware"
)

type IncomingMessageHandler interface {
	HandleMessage(client *Client, msg *WSMessage)
	Shutdown()
}

type incomingMessageHandler struct {
	idempotency           *IdempotencyTracker
	idempotencyMiddleware *middleware.IdempotencyMiddleware
	processor             *MessageProcessor
}

func NewIncomingMessageHandler(idempotency *IdempotencyTracker, idempotencyMiddleware *middleware.IdempotencyMiddleware, processor *MessageProcessor) IncomingMessageHandler {
	return &incomingMessageHandler{
		idempotency:           idempotency,
		idempotencyMiddleware: idempotencyMiddleware,
		processor:             processor,
	}
}

func (h *incomingMessageHandler) HandleMessage(client *Client, msg *WSMessage) {
	handler := func(ctx context.Context, c middleware.Client, m *middleware.WSMessage) error {
		h.processor.Submit(ctx, client, msg)
		return nil
	}

	middlewareMsg := &middleware.WSMessage{
		Type:    string(msg.Type),
		Payload: msg.Payload,
	}

	switch msg.Type {
	case TypeEphemeralKey:
		if err := h.idempotencyMiddleware.Handle(client.ctx, client, middlewareMsg, handler); err != nil {
			return
		}
		return

	case TypeMessage:
		var payload MessagePayload
		if err := json.Unmarshal(msg.Payload, &payload); err == nil && payload.MessageID != "" {
			operationID := h.idempotency.GenerateOperationID(client.userID+":"+payload.MessageID, msg.Type, msg.Payload)
			if err := h.idempotencyMiddleware.HandleWithOperationID(client.ctx, client, middlewareMsg, operationID, handler); err != nil {
				return
			}
			return
		}

	case TypeFileChunk:
		var payload FileChunkPayload
		if err := json.Unmarshal(msg.Payload, &payload); err == nil && payload.FileID != "" {
			chunkKey := client.userID + ":" + payload.FileID + ":" + strconv.Itoa(payload.ChunkIndex)
			operationID := h.idempotency.GenerateOperationID(chunkKey, msg.Type, msg.Payload)
			if err := h.idempotencyMiddleware.HandleWithOperationID(client.ctx, client, middlewareMsg, operationID, handler); err != nil {
				return
			}
			return
		}
	}

	h.processor.Submit(client.ctx, client, msg)
}

func (h *incomingMessageHandler) Shutdown() {
	if h.idempotency != nil {
		h.idempotency.Shutdown()
	}
	if h.processor != nil {
		h.processor.Shutdown()
	}
}
