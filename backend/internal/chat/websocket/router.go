package websocket

import (
	"context"
	"encoding/json"

	"github.com/AlibekovAA/dh-secure-chat/backend/internal/chat/metrics"
	commonerrors "github.com/AlibekovAA/dh-secure-chat/backend/internal/common/errors"
	"github.com/AlibekovAA/dh-secure-chat/backend/internal/common/logger"
)

type MessageRouter interface {
	Route(ctx context.Context, client *Client, msg *WSMessage) error
}

type messageRouter struct {
	hub       *Hub
	validator MessageValidator
	log       *logger.Logger
}

func NewMessageRouter(hub *Hub, validator MessageValidator, log *logger.Logger) MessageRouter {
	return &messageRouter{
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

func (r *messageRouter) Route(ctx context.Context, client *Client, msg *WSMessage) error {
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
			r.log.WithFields(ctx, logger.Fields{
				"user_id": client.userID,
				"type":    "file_chunk",
				"action":  "ws_invalid_payload",
			}).Warnf("websocket invalid file_chunk payload: %v", err)
			metrics.IncrementWebSocketError("invalid_file_chunk_payload")
			return commonerrors.ErrInvalidPayload.WithCause(err)
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
			r.log.WithFields(ctx, logger.Fields{
				"user_id": client.userID,
				"type":    "file_complete",
				"action":  "ws_invalid_payload",
			}).Warnf("websocket invalid file_complete payload: %v", err)
			return commonerrors.ErrInvalidPayload.WithCause(err)
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
		r.log.WithFields(ctx, logger.Fields{
			"user_id": client.userID,
			"type":    string(msg.Type),
			"action":  "ws_unknown_message_type",
		}).Warn("websocket unknown message type")
		metrics.IncrementWebSocketError("unknown_message_type")
		return commonerrors.ErrUnknownMessageType
	}
}

func (r *messageRouter) routeSimple(ctx context.Context, client *Client, msg *WSMessage, payload payloadWithTo, msgType string, requireOnline bool, fromUserID string) error {
	if err := json.Unmarshal(msg.Payload, payload); err != nil {
		r.log.WithFields(ctx, logger.Fields{
			"user_id": client.userID,
			"type":    msgType,
			"action":  "ws_invalid_payload",
		}).Warnf("websocket invalid payload: %v", err)
		return commonerrors.ErrInvalidPayload.WithCause(err)
	}

	if r.hub.forwardMessage(ctx, msg, payload, requireOnline, fromUserID) {
		metrics.IncrementWebSocketMessage(msgType)
	}
	return nil
}

func (r *messageRouter) routeWithModifiedPayload(ctx context.Context, client *Client, msg *WSMessage, payload payloadWithTo, msgType string, requireOnline bool) error {
	if err := json.Unmarshal(msg.Payload, payload); err != nil {
		r.log.WithFields(ctx, logger.Fields{
			"user_id": client.userID,
			"type":    msgType,
			"action":  "ws_invalid_payload",
		}).Warnf("websocket invalid payload: %v", err)
		metrics.IncrementWebSocketError("invalid_" + msgType + "_payload")
		return commonerrors.ErrInvalidPayload.WithCause(err)
	}

	if p, ok := payload.(payloadWithFrom); ok {
		p.SetFrom(client.userID)
	}

	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		r.log.WithFields(ctx, logger.Fields{
			"user_id": client.userID,
			"type":    msgType,
			"action":  "ws_marshal_failed",
		}).Warnf("websocket failed to marshal payload: %v", err)
		return commonerrors.ErrMarshalError.WithCause(err)
	}

	msg.Payload = payloadBytes
	if r.hub.forwardMessage(ctx, msg, payload, requireOnline, client.userID) {
		metrics.IncrementWebSocketMessage(msgType)
	}
	return nil
}

func (r *messageRouter) routeFileStart(ctx context.Context, client *Client, msg *WSMessage) error {
	var payload FileStartPayload
	if err := json.Unmarshal(msg.Payload, &payload); err != nil {
		r.log.WithFields(ctx, logger.Fields{
			"user_id": client.userID,
			"type":    "file_start",
			"action":  "ws_invalid_payload",
		}).Warnf("websocket invalid file_start payload: %v", err)
		metrics.IncrementWebSocketError("invalid_file_start_payload")
		return commonerrors.ErrInvalidPayload.WithCause(err)
	}

	if err := r.validator.ValidateFileStart(payload); err != nil {
		r.log.WithFields(ctx, logger.Fields{
			"user_id": client.userID,
			"file_id": payload.FileID,
			"action":  "ws_file_validation_failed",
		}).Warnf("websocket file_start validation failed: %v", err)
		metrics.IncrementWebSocketError("file_validation_failed")
		return err
	}

	payload.From = client.userID
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		r.log.WithFields(ctx, logger.Fields{
			"user_id": client.userID,
			"file_id": payload.FileID,
			"action":  "ws_file_marshal_failed",
		}).Warnf("websocket failed to marshal file_start payload: %v", err)
		return commonerrors.ErrMarshalError.WithCause(err)
	}

	forwardMsg := &WSMessage{
		Type:    msg.Type,
		Payload: payloadBytes,
	}

	r.log.WithFields(ctx, logger.Fields{
		"from":     client.userID,
		"to":       payload.To,
		"file_id":  payload.FileID,
		"filename": payload.Filename,
		"action":   "ws_file_start",
	}).Debug("websocket file_start")

	if r.hub.forwardMessage(ctx, forwardMsg, &payload, true, client.userID) {
		metrics.IncrementWebSocketFile()
		metrics.IncrementWebSocketMessage("file_start")
		r.hub.trackFileTransfer(payload)
	}

	return nil
}
