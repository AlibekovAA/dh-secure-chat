package websocket

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/AlibekovAA/dh-secure-chat/backend/internal/chat/metrics"
	"github.com/AlibekovAA/dh-secure-chat/backend/internal/common/logger"
)

type MessageRouter struct {
	hub       *Hub
	validator MessageValidator
	log       *logger.Logger
}

func NewMessageRouter(hub *Hub, validator MessageValidator, log *logger.Logger) *MessageRouter {
	return &MessageRouter{
		hub:       hub,
		validator: validator,
		log:       log,
	}
}

type payloadWithFrom interface {
	SetFrom(from string)
}

func (p *EphemeralKeyPayload) SetFrom(from string)  { p.From = from }
func (p *FileStartPayload) SetFrom(from string)     { p.From = from }
func (p *FileChunkPayload) SetFrom(from string)     { p.From = from }
func (p *FileCompletePayload) SetFrom(from string)  { p.From = from }
func (p *TypingPayload) SetFrom(from string)        { p.From = from }
func (p *ReactionPayload) SetFrom(from string)      { p.From = from }
func (p *MessagePayload) SetFrom(from string)       { p.From = from }
func (p *MessageDeletePayload) SetFrom(from string) { p.From = from }

func (r *MessageRouter) Route(ctx context.Context, client *Client, msg *WSMessage) error {
	switch msg.Type {
	case TypeEphemeralKey:
		return r.routeWithModifiedPayload(ctx, client, msg, &EphemeralKeyPayload{}, "ephemeral_key", true)

	case TypeMessage:
		return r.routeWithModifiedPayload(ctx, client, msg, &MessagePayload{}, "message", true)

	case TypeSessionEstablished:
		return r.routeSimple(ctx, client, msg, &SessionEstablishedPayload{}, "session_established", false, client.userID)

	case TypeFileStart:
		return r.routeFileStart(ctx, client, msg)

	case TypeFileChunk:
		var payload FileChunkPayload
		if err := json.Unmarshal(msg.Payload, &payload); err != nil {
			r.log.Warnf("websocket invalid file_chunk payload user_id=%s: %v", client.userID, err)
			metrics.IncrementWebSocketError("invalid_file_chunk_payload")
			return fmt.Errorf("invalid file_chunk payload: %w", err)
		}
		payload.From = client.userID
		payloadBytes, _ := json.Marshal(payload)
		msg.Payload = payloadBytes
		if r.hub.forwardMessage(ctx, msg, &payload, true, client.userID) {
			metrics.IncrementWebSocketFileChunk()
			metrics.IncrementWebSocketMessage("file_chunk")
			r.hub.updateFileTransferProgress(payload.FileID, payload.ChunkIndex)
		}
		return nil

	case TypeFileComplete:
		var payload FileCompletePayload
		if err := json.Unmarshal(msg.Payload, &payload); err != nil {
			r.log.Warnf("websocket invalid file_complete payload user_id=%s: %v", client.userID, err)
			return fmt.Errorf("invalid file_complete payload: %w", err)
		}
		payload.From = client.userID
		payloadBytes, _ := json.Marshal(payload)
		msg.Payload = payloadBytes
		if r.hub.forwardMessage(ctx, msg, &payload, true, client.userID) {
			metrics.IncrementWebSocketMessage("file_complete")
			r.hub.completeFileTransfer(payload.FileID)
		}
		return nil

	case TypeAck:
		return r.routeSimple(ctx, client, msg, &AckPayload{}, "ack", false, client.userID)

	case TypeTyping:
		return r.routeWithModifiedPayload(ctx, client, msg, &TypingPayload{}, "typing", true)

	case TypeReaction:
		return r.routeWithModifiedPayload(ctx, client, msg, &ReactionPayload{}, "reaction", true)

	case TypeMessageDelete:
		return r.routeWithModifiedPayload(ctx, client, msg, &MessageDeletePayload{}, "message_delete", true)

	default:
		r.log.Warnf("websocket unknown message type user_id=%s type=%s", client.userID, msg.Type)
		metrics.IncrementWebSocketError("unknown_message_type")
		return fmt.Errorf("unknown message type: %s", msg.Type)
	}
}

func (r *MessageRouter) routeSimple(ctx context.Context, client *Client, msg *WSMessage, payload payloadWithTo, msgType string, requireOnline bool, fromUserID string) error {
	if err := json.Unmarshal(msg.Payload, payload); err != nil {
		r.log.Warnf("websocket invalid %s payload user_id=%s: %v", msgType, client.userID, err)
		return fmt.Errorf("invalid %s payload: %w", msgType, err)
	}

	if r.hub.forwardMessage(ctx, msg, payload, requireOnline, fromUserID) {
		metrics.IncrementWebSocketMessage(msgType)
	}
	return nil
}

func (r *MessageRouter) routeWithModifiedPayload(ctx context.Context, client *Client, msg *WSMessage, payload payloadWithTo, msgType string, requireOnline bool) error {
	if err := json.Unmarshal(msg.Payload, payload); err != nil {
		r.log.Warnf("websocket invalid %s payload user_id=%s: %v", msgType, client.userID, err)
		metrics.IncrementWebSocketError("invalid_" + msgType + "_payload")
		return fmt.Errorf("invalid %s payload: %w", msgType, err)
	}

	if p, ok := payload.(payloadWithFrom); ok {
		p.SetFrom(client.userID)
	}

	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		r.log.Warnf("websocket failed to marshal %s payload user_id=%s: %v", msgType, client.userID, err)
		return fmt.Errorf("failed to marshal payload: %w", err)
	}

	msg.Payload = payloadBytes
	if r.hub.forwardMessage(ctx, msg, payload, requireOnline, client.userID) {
		metrics.IncrementWebSocketMessage(msgType)
	}
	return nil
}

func (r *MessageRouter) routeFileStart(ctx context.Context, client *Client, msg *WSMessage) error {
	var payload FileStartPayload
	if err := json.Unmarshal(msg.Payload, &payload); err != nil {
		r.log.Warnf("websocket invalid file_start payload user_id=%s: %v", client.userID, err)
		metrics.IncrementWebSocketError("invalid_file_start_payload")
		return fmt.Errorf("invalid file_start payload: %w", err)
	}

	if err := r.validator.ValidateFileStart(payload); err != nil {
		r.log.Warnf("websocket file_start validation failed user_id=%s: %v", client.userID, err)
		metrics.IncrementWebSocketError("file_validation_failed")
		return fmt.Errorf("file validation failed: %w", err)
	}

	payload.From = client.userID
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		r.log.Warnf("websocket failed to marshal file_start payload user_id=%s: %v", client.userID, err)
		return fmt.Errorf("failed to marshal payload: %w", err)
	}

	forwardMsg := &WSMessage{
		Type:    msg.Type,
		Payload: payloadBytes,
	}

	r.log.Debugf("websocket file_start from=%s to=%s file_id=%s filename=%s", client.userID, payload.To, payload.FileID, payload.Filename)

	if r.hub.forwardMessage(ctx, forwardMsg, &payload, true, client.userID) {
		metrics.IncrementWebSocketFile()
		metrics.IncrementWebSocketMessage("file_start")
		r.hub.trackFileTransfer(payload)
	}

	return nil
}
