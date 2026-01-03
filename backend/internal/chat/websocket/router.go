package websocket

import (
	"context"
	"encoding/json"

	commonerrors "github.com/AlibekovAA/dh-secure-chat/backend/internal/common/errors"
	commonhttp "github.com/AlibekovAA/dh-secure-chat/backend/internal/common/http"
	"github.com/AlibekovAA/dh-secure-chat/backend/internal/common/logger"
	observabilitymetrics "github.com/AlibekovAA/dh-secure-chat/backend/internal/observability/metrics"
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

func (r *messageRouter) handleUnmarshalError(ctx context.Context, client *Client, err error, msgType string) error {
	if err == nil {
		return nil
	}
	wsErr := commonerrors.ErrInvalidPayload.WithCause(err)
	r.log.WithFields(ctx, logger.Fields{
		"user_id": client.userID,
		"type":    msgType,
		"action":  "ws_invalid_payload",
	}).Warnf("websocket invalid payload: %v", err)
	observabilitymetrics.ChatWebSocketErrors.WithLabelValues("invalid_" + msgType + "_payload").Inc()
	r.hub.sendErrorToUser(client.userID, wsErr)
	return wsErr
}

func (r *messageRouter) handleValidateUserIDError(ctx context.Context, client *Client, userID, msgType string) error {
	if err := commonhttp.ValidateUUID(userID); err != nil {
		wsErr := commonerrors.ErrInvalidPayload.WithCause(err)
		r.log.WithFields(ctx, logger.Fields{
			"user_id": client.userID,
			"to":      userID,
			"type":    msgType,
			"action":  "ws_invalid_user_id",
		}).Warnf("websocket invalid user ID: %v", err)
		observabilitymetrics.ChatWebSocketErrors.WithLabelValues("invalid_user_id").Inc()
		r.hub.sendErrorToUser(client.userID, wsErr)
		return wsErr
	}
	return nil
}

func (r *messageRouter) handleMarshalError(ctx context.Context, client *Client, err error, msgType string) error {
	if err == nil {
		return nil
	}
	wsErr := commonerrors.ErrMarshalError.WithCause(err)
	r.log.WithFields(ctx, logger.Fields{
		"user_id": client.userID,
		"type":    msgType,
		"action":  "ws_marshal_failed",
	}).Warnf("websocket failed to marshal payload: %v", err)
	observabilitymetrics.ChatWebSocketErrors.WithLabelValues("marshal_failed").Inc()
	return wsErr
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
func (p *MessageEditPayload) SetFrom(from string)   { p.From = from }
func (p *MessageReadPayload) SetFrom(from string)   { p.From = from }

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
		return r.routeFileChunk(ctx, client, msg)

	case TypeFileComplete:
		return r.routeFileComplete(ctx, client, msg)

	case TypeAck:
		return r.routeSimple(ctx, client, msg, &AckPayload{}, "ack", false, client.userID)

	case TypeTyping:
		return r.routeWithModifiedPayload(ctx, client, msg, &TypingPayload{}, "typing", true)

	case TypeReaction:
		return r.routeWithModifiedPayload(ctx, client, msg, &ReactionPayload{}, "reaction", true)

	case TypeMessageDelete:
		return r.routeWithModifiedPayload(ctx, client, msg, &MessageDeletePayload{}, "message_delete", true)

	case TypeMessageEdit:
		return r.routeWithModifiedPayload(ctx, client, msg, &MessageEditPayload{}, "message_edit", true)

	case TypeMessageRead:
		return r.routeWithModifiedPayload(ctx, client, msg, &MessageReadPayload{}, "message_read", true)

	default:
		r.log.WithFields(ctx, logger.Fields{
			"user_id": client.userID,
			"type":    string(msg.Type),
			"action":  "ws_unknown_message_type",
		}).Warn("websocket unknown message type")
		observabilitymetrics.ChatWebSocketErrors.WithLabelValues("unknown_message_type").Inc()
		r.hub.sendErrorToUser(client.userID, commonerrors.ErrUnknownMessageType)
		return commonerrors.ErrUnknownMessageType
	}
}

func (r *messageRouter) routeSimple(ctx context.Context, client *Client, msg *WSMessage, payload payloadWithTo, msgType string, requireOnline bool, fromUserID string) error {
	return r.routePayload(ctx, client, msg, payload, msgType, requireOnline, fromUserID, false)
}

func (r *messageRouter) routeWithModifiedPayload(ctx context.Context, client *Client, msg *WSMessage, payload payloadWithTo, msgType string, requireOnline bool) error {
	return r.routePayload(ctx, client, msg, payload, msgType, requireOnline, client.userID, true)
}

func (r *messageRouter) routePayload(ctx context.Context, client *Client, msg *WSMessage, payload payloadWithTo, msgType string, requireOnline bool, fromUserID string, modifyPayload bool) error {
	if err := json.Unmarshal(msg.Payload, payload); err != nil {
		return r.handleUnmarshalError(ctx, client, err, msgType)
	}

	to := payload.GetTo()
	if err := r.handleValidateUserIDError(ctx, client, to, msgType); err != nil {
		return err
	}

	if modifyPayload {
		if p, ok := payload.(payloadWithFrom); ok {
			p.SetFrom(client.userID)
		}

		payloadBytes, err := json.Marshal(payload)
		if err != nil {
			return r.handleMarshalError(ctx, client, err, msgType)
		}
		msg.Payload = payloadBytes
	}

	if r.hub.forwardMessage(ctx, msg, payload, requireOnline, fromUserID) {
		observabilitymetrics.ChatWebSocketMessagesTotal.WithLabelValues(msgType).Inc()
	}
	return nil
}

func (r *messageRouter) routeFileChunk(ctx context.Context, client *Client, msg *WSMessage) error {
	var payload FileChunkPayload
	if err := r.unmarshalAndValidate(ctx, client, msg, &payload, "file_chunk"); err != nil {
		return err
	}

	payload.From = client.userID
	if err := r.marshalAndForward(ctx, client, msg, &payload, "file_chunk", true); err != nil {
		return err
	}

	observabilitymetrics.ChatWebSocketFilesChunksTotal.Inc()
	r.hub.updateFileTransferProgress(payload.FileID, payload.ChunkIndex)
	return nil
}

func (r *messageRouter) routeFileComplete(ctx context.Context, client *Client, msg *WSMessage) error {
	var payload FileCompletePayload
	if err := r.unmarshalAndValidate(ctx, client, msg, &payload, "file_complete"); err != nil {
		return err
	}

	payload.From = client.userID
	if err := r.marshalAndForward(ctx, client, msg, &payload, "file_complete", true); err != nil {
		return err
	}

	r.hub.completeFileTransfer(payload.FileID)
	return nil
}

func (r *messageRouter) unmarshalAndValidate(ctx context.Context, client *Client, msg *WSMessage, payload payloadWithTo, msgType string) error {
	if err := json.Unmarshal(msg.Payload, payload); err != nil {
		return r.handleUnmarshalError(ctx, client, err, msgType)
	}
	return r.handleValidateUserIDError(ctx, client, payload.GetTo(), msgType)
}

func (r *messageRouter) marshalAndForward(ctx context.Context, client *Client, msg *WSMessage, payload payloadWithTo, msgType string, requireOnline bool) error {
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return r.handleMarshalError(ctx, client, err, msgType)
	}

	msg.Payload = payloadBytes
	if r.hub.forwardMessage(ctx, msg, payload, requireOnline, client.userID) {
		observabilitymetrics.ChatWebSocketMessagesTotal.WithLabelValues(msgType).Inc()
	}
	return nil
}

func (r *messageRouter) routeFileStart(ctx context.Context, client *Client, msg *WSMessage) error {
	var payload FileStartPayload
	if err := r.unmarshalAndValidate(ctx, client, msg, &payload, "file_start"); err != nil {
		return err
	}

	if err := r.validator.ValidateFileStart(payload); err != nil {
		wsErr := commonerrors.ErrFileSizeExceeded
		if de, ok := commonerrors.AsDomainError(err); ok {
			wsErr = de
		}
		r.log.WithFields(ctx, logger.Fields{
			"user_id": client.userID,
			"file_id": payload.FileID,
			"action":  "ws_file_validation_failed",
		}).Warnf("websocket file_start validation failed: %v", err)
		observabilitymetrics.ChatWebSocketErrors.WithLabelValues("file_validation_failed").Inc()
		r.hub.sendErrorToUser(client.userID, wsErr)
		return wsErr
	}

	payload.From = client.userID
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return r.handleMarshalError(ctx, client, err, "file_start")
	}

	forwardMsg := &WSMessage{Type: msg.Type, Payload: payloadBytes}
	r.log.WithFields(ctx, logger.Fields{
		"from":     client.userID,
		"to":       payload.To,
		"file_id":  payload.FileID,
		"filename": payload.Filename,
		"action":   "ws_file_start",
	}).Debug("websocket file_start")

	if r.hub.forwardMessage(ctx, forwardMsg, &payload, true, client.userID) {
		observabilitymetrics.ChatWebSocketFilesTotal.Inc()
		observabilitymetrics.ChatWebSocketMessagesTotal.WithLabelValues("file_start").Inc()
		r.hub.trackFileTransfer(payload)
	}
	return nil
}
