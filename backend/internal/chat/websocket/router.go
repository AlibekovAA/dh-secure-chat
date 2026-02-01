package websocket

import (
	"context"
	"encoding/json"
	"errors"

	commonerrors "github.com/AlibekovAA/dh-secure-chat/backend/internal/common/errors"
	commonhttp "github.com/AlibekovAA/dh-secure-chat/backend/internal/common/http"
	"github.com/AlibekovAA/dh-secure-chat/backend/internal/common/logger"
	observabilitymetrics "github.com/AlibekovAA/dh-secure-chat/backend/internal/observability/metrics"
)

type MessageRouter interface {
	Route(ctx context.Context, client *Client, msg *WSMessage) error
}

type messageRouter struct {
	sender          MessageSender
	presence        *PresenceService
	fileService     *FileTransferService
	validator       MessageValidator
	log             *logger.Logger
	debugSampleRate float64
}

func NewMessageRouter(sender MessageSender, presence *PresenceService, fileService *FileTransferService, validator MessageValidator, log *logger.Logger, debugSampleRate float64) MessageRouter {
	return &messageRouter{
		sender:          sender,
		presence:        presence,
		fileService:     fileService,
		validator:       validator,
		log:             log,
		debugSampleRate: debugSampleRate,
	}
}

type payloadWithTo interface {
	GetTo() string
}

func (p EphemeralKeyPayload) GetTo() string       { return p.To }
func (p MessagePayload) GetTo() string            { return p.To }
func (p SessionEstablishedPayload) GetTo() string { return p.To }
func (p FileStartPayload) GetTo() string          { return p.To }
func (p FileChunkPayload) GetTo() string          { return p.To }
func (p FileCompletePayload) GetTo() string       { return p.To }
func (p AckPayload) GetTo() string                { return p.To }
func (p TypingPayload) GetTo() string             { return p.To }
func (p ReactionPayload) GetTo() string           { return p.To }
func (p MessageDeletePayload) GetTo() string      { return p.To }
func (p MessageEditPayload) GetTo() string        { return p.To }
func (p MessageReadPayload) GetTo() string        { return p.To }

type errorHandlerConfig struct {
	err              commonerrors.DomainError
	action           string
	metricLabel      string
	sendToUser       bool
	logMessage       string
	additionalFields logger.Fields
}

func (r *messageRouter) handleError(ctx context.Context, client *Client, err error, msgType string, config errorHandlerConfig) error {
	if err == nil {
		return nil
	}

	wsErr := config.err.WithCause(err)
	fields := logger.Fields{
		"user_id": client.userID,
		"type":    msgType,
		"action":  config.action,
	}
	for k, v := range config.additionalFields {
		fields[k] = v
	}

	r.log.WithFields(ctx, fields).Warnf(config.logMessage, err)
	observabilitymetrics.ChatWebSocketErrors.WithLabelValues(config.metricLabel).Inc()

	if config.sendToUser {
		r.sender.SendErrorToUser(client.userID, wsErr)
	}

	return wsErr
}

func (r *messageRouter) handleUnmarshalError(ctx context.Context, client *Client, err error, msgType string) error {
	return r.handleError(ctx, client, err, msgType, errorHandlerConfig{
		err:         commonerrors.ErrInvalidPayload,
		action:      "ws_invalid_payload",
		metricLabel: "invalid_" + msgType + "_payload",
		sendToUser:  true,
		logMessage:  "websocket invalid payload: %v",
	})
}

func (r *messageRouter) handleValidateUserIDError(ctx context.Context, client *Client, userID, msgType string) error {
	if err := commonhttp.ValidateUUID(userID); err != nil {
		return r.handleError(ctx, client, err, msgType, errorHandlerConfig{
			err:              commonerrors.ErrInvalidPayload,
			action:           "ws_invalid_user_id",
			metricLabel:      "invalid_user_id",
			sendToUser:       true,
			logMessage:       "websocket invalid user ID: %v",
			additionalFields: logger.Fields{"to": userID},
		})
	}
	return nil
}

func (r *messageRouter) handleMarshalError(ctx context.Context, client *Client, err error, msgType string) error {
	return r.handleError(ctx, client, err, msgType, errorHandlerConfig{
		err:         commonerrors.ErrMarshalError,
		action:      "ws_marshal_failed",
		metricLabel: "marshal_failed",
		sendToUser:  false,
		logMessage:  "websocket failed to marshal payload: %v",
	})
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
		r.sender.SendErrorToUser(client.userID, commonerrors.ErrUnknownMessageType)
		return commonerrors.ErrUnknownMessageType
	}
}

func (r *messageRouter) forwardMessage(ctx context.Context, msg *WSMessage, payload payloadWithTo, requireOnline bool, fromUserID string) bool {
	to := payload.GetTo()
	if to == "" {
		if fromUserID != "" {
			r.log.WithFields(ctx, logger.Fields{
				"user_id": fromUserID,
				"type":    string(msg.Type),
				"action":  "ws_message_missing_to",
			}).Warn("websocket message missing 'to' field")
		}
		return false
	}

	if fromUserID != "" && to == fromUserID {
		r.log.WithFields(ctx, logger.Fields{
			"user_id": fromUserID,
			"type":    string(msg.Type),
			"action":  "ws_message_to_self",
		}).Warn("websocket message to self")
		return false
	}

	if requireOnline && !r.sender.IsUserOnline(to) {
		if fromUserID != "" {
			if err := r.presence.SendPeerOffline(ctx, fromUserID, to); err != nil {
				r.log.WithFields(ctx, logger.Fields{
					"from":   fromUserID,
					"to":     to,
					"action": "ws_peer_offline_send",
				}).Errorf("websocket failed to send peer_offline: %v", err)
			}
		}
		r.log.WithFields(ctx, logger.Fields{
			"from":   fromUserID,
			"to":     to,
			"type":   string(msg.Type),
			"action": "ws_message_offline",
		}).Info("websocket message to offline user")
		return false
	}

	if !requireOnline {
		exists, err := r.presence.CheckUserExists(ctx, to)
		if err != nil {
			r.log.WithFields(ctx, logger.Fields{
				"from":   fromUserID,
				"to":     to,
				"type":   string(msg.Type),
				"action": "ws_user_check_failed",
			}).Errorf("websocket failed to check user existence: %v", err)
			return false
		}
		if !exists {
			if fromUserID != "" {
				r.log.WithFields(ctx, logger.Fields{
					"from":   fromUserID,
					"to":     to,
					"type":   string(msg.Type),
					"action": "ws_message_user_not_found",
				}).Warn("websocket message to non-existent user")
			}
			return false
		}
	}

	if err := r.sender.SendToUserWithContext(ctx, to, msg); err != nil {
		if !errors.Is(err, context.Canceled) && !errors.Is(err, context.DeadlineExceeded) {
			r.log.WithFields(ctx, logger.Fields{
				"from":   fromUserID,
				"to":     to,
				"type":   string(msg.Type),
				"action": "ws_forward_failed",
			}).Warnf("websocket failed to forward message: %v", err)
		}
		return false
	}

	if r.log.ShouldLog(logger.DEBUG) && r.log.ShouldSample(r.debugSampleRate) {
		r.log.WithFields(ctx, logger.Fields{
			"from":   fromUserID,
			"to":     to,
			"type":   string(msg.Type),
			"action": "ws_message_forwarded",
		}).Debug("websocket message forwarded")
	}
	return true
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

	if r.forwardMessage(ctx, msg, payload, requireOnline, fromUserID) {
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
	r.fileService.UpdateProgress(payload.FileID, payload.ChunkIndex)
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

	r.fileService.Complete(payload.FileID)
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
	if r.forwardMessage(ctx, msg, payload, requireOnline, client.userID) {
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
		r.sender.SendErrorToUser(client.userID, wsErr)
		return wsErr
	}

	payload.From = client.userID
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return r.handleMarshalError(ctx, client, err, "file_start")
	}

	forwardMsg := &WSMessage{Type: msg.Type, Payload: payloadBytes}
	if r.log.ShouldLog(logger.DEBUG) {
		r.log.WithFields(ctx, logger.Fields{
			"from":     client.userID,
			"to":       payload.To,
			"file_id":  payload.FileID,
			"filename": payload.Filename,
			"action":   "ws_file_start",
		}).Debug("websocket file_start")
	}

	if r.forwardMessage(ctx, forwardMsg, &payload, true, client.userID) {
		observabilitymetrics.ChatWebSocketFilesTotal.Inc()
		observabilitymetrics.ChatWebSocketMessagesTotal.WithLabelValues("file_start").Inc()
		r.fileService.Track(payload)
	}
	return nil
}
