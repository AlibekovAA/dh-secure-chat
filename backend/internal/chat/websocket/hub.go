package websocket

import (
	"bytes"
	"context"
	"encoding/json"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/AlibekovAA/dh-secure-chat/backend/internal/common/clock"
	"github.com/AlibekovAA/dh-secure-chat/backend/internal/common/constants"
	commonerrors "github.com/AlibekovAA/dh-secure-chat/backend/internal/common/errors"
	"github.com/AlibekovAA/dh-secure-chat/backend/internal/common/logger"
	observabilitymetrics "github.com/AlibekovAA/dh-secure-chat/backend/internal/observability/metrics"
)

var validAudioMimeTypes = map[string]bool{
	"audio/webm":  true,
	"audio/ogg":   true,
	"audio/mpeg":  true,
	"audio/mp4":   true,
	"audio/wav":   true,
	"audio/x-m4a": true,
}

func normalizeAudioMimeType(mimeType string) string {
	mimeType = strings.ToLower(strings.TrimSpace(mimeType))
	if idx := strings.Index(mimeType, ";"); idx != -1 {
		mimeType = mimeType[:idx]
	}
	return strings.TrimSpace(mimeType)
}

func isValidAudioMimeType(mimeType string) bool {
	normalized := normalizeAudioMimeType(mimeType)
	return validAudioMimeTypes[normalized]
}

type jsonEncoderPoolItem struct {
	buf *bytes.Buffer
	enc *json.Encoder
}

var jsonEncoderPool = sync.Pool{
	New: func() interface{} {
		buf := &bytes.Buffer{}
		return &jsonEncoderPoolItem{
			buf: buf,
			enc: json.NewEncoder(buf),
		}
	},
}

type Hub struct {
	clients        sync.Map
	register       chan *Client
	unregister     chan *Client
	clientCount    atomic.Int64
	maxConnections int
	log            *logger.Logger
	sendTimeout    time.Duration
	clock          clock.Clock
	ctx            context.Context
	cancel         context.CancelFunc

	messageHandler  IncomingMessageHandler
	presenceService *PresenceService
	fileService     *FileTransferService
}

type HubDeps struct {
	Log   *logger.Logger
	Clock clock.Clock
}

type HubConfig struct {
	SendTimeout             time.Duration
	MaxConnections          int
	MaxFileSize             int64
	MaxVoiceSize            int64
	ProcessorWorkers        int
	ProcessorQueueSize      int
	LastSeenUpdateInterval  time.Duration
	CircuitBreakerThreshold int32
	CircuitBreakerTimeout   time.Duration
	CircuitBreakerReset     time.Duration
	FileTransferTimeout     time.Duration
	IdempotencyTTL          time.Duration
	DebugSampleRate         float64
}

func NewHub(deps HubDeps, config HubConfig) *Hub {
	ctx, cancel := context.WithCancel(context.Background())
	timeClock := deps.Clock
	if timeClock == nil {
		timeClock = clock.NewRealClock()
	}
	return &Hub{
		register:       make(chan *Client),
		unregister:     make(chan *Client),
		log:            deps.Log,
		sendTimeout:    config.SendTimeout,
		maxConnections: config.MaxConnections,
		clock:          timeClock,
		ctx:            ctx,
		cancel:         cancel,
	}
}

func (h *Hub) Context() context.Context {
	return h.ctx
}

func (h *Hub) Wire(messageHandler IncomingMessageHandler, presenceService *PresenceService, fileService *FileTransferService) {
	h.messageHandler = messageHandler
	h.presenceService = presenceService
	h.fileService = fileService
	go presenceService.StartCleanup()
	go fileService.StartCleanup()
}

func (h *Hub) Register(client *Client) {
	h.register <- client
}

func (h *Hub) Unregister(client *Client) {
	h.unregister <- client
}

func (h *Hub) Run(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			h.shutdown()
			return

		case client := <-h.register:
			if existing, ok := h.clients.Load(client.userID); ok {
				existingClient := existing.(*Client)
				h.log.WithFields(client.ctx, logger.Fields{
					"user_id":  existingClient.userID,
					"username": existingClient.username,
					"action":   "ws_close_existing",
				}).Info("websocket closing existing connection")
				existingClient.Stop()
				existingClient.Close()
				existingClient.WaitForShutdown(constants.WebSocketClientShutdownTimeout)
				h.clients.Delete(client.userID)
				observabilitymetrics.ChatWebSocketConnectionsActive.Dec()
				h.clientCount.Add(-1)
			}

			currentCount := int(h.clientCount.Load())
			if currentCount >= h.maxConnections {
				observabilitymetrics.ChatWebSocketConnectionsRejected.Inc()
				h.log.WithFields(client.ctx, logger.Fields{
					"user_id": client.userID,
					"current": currentCount,
					"max":     h.maxConnections,
					"action":  "ws_register_rejected",
				}).Warn("websocket connection rejected: max connections limit reached")
				client.Stop()
				client.Close()
				client.WaitForShutdown(constants.WebSocketClientShutdownTimeout)
				client.conn.Close()
				continue
			}

			h.clients.Store(client.userID, client)
			totalClients := h.clientCount.Add(1)
			observabilitymetrics.ChatWebSocketConnectionsActive.Inc()
			h.log.WithFields(client.ctx, logger.Fields{
				"user_id":  client.userID,
				"username": client.username,
				"total":    totalClients,
				"action":   "ws_register",
			}).Info("websocket client registered")
			if h.presenceService != nil {
				h.presenceService.UpdateLastSeenDebounced(client.userID)
			}

		case client := <-h.unregister:
			h.handleUnregister(client)
		}
	}
}

func (h *Hub) shutdown() {
	clients := make([]*Client, 0)
	h.clients.Range(func(key, value interface{}) bool {
		clients = append(clients, value.(*Client))
		return true
	})

	shutdownMsg, err := json.Marshal(&WSMessage{Type: "shutdown"})
	if err != nil {
		h.log.WithFields(h.ctx, logger.Fields{
			"action": "ws_shutdown_marshal",
		}).Errorf("websocket failed to marshal shutdown message: %v", err)
	}

	for _, client := range clients {
		client.Stop()
		if err == nil {
			ctx, cancel := context.WithTimeout(context.Background(), constants.WebSocketShutdownNotificationTimeout)
			select {
			case client.send <- shutdownMsg:
			case <-ctx.Done():
				h.log.WithFields(ctx, logger.Fields{
					"user_id": client.userID,
					"action":  "ws_shutdown_timeout",
				}).Warn("websocket shutdown notification timeout")
			}
			cancel()
		}
		client.Close()
		client.WaitForShutdown(constants.WebSocketClientShutdownTimeoutLong)
	}

	h.clients.Range(func(key, value interface{}) bool {
		h.clients.Delete(key)
		return true
	})

	h.log.WithFields(h.ctx, logger.Fields{
		"clients": len(clients),
		"action":  "ws_hub_shutdown",
	}).Info("websocket hub shutdown completed")
}

func (h *Hub) SendToUser(userID string, message *WSMessage) bool {
	err := h.SendToUserWithContext(h.ctx, userID, message)
	return err == nil
}

func (h *Hub) SendToUserWithContext(ctx context.Context, userID string, message *WSMessage) error {
	startTime := h.clock.Now()
	defer func() {
		duration := h.clock.Since(startTime).Seconds()
		observabilitymetrics.ChatWebSocketMessageSendDurationSeconds.WithLabelValues(string(message.Type)).Observe(duration)
	}()

	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	value, ok := h.clients.Load(userID)
	if !ok {
		return commonerrors.ErrUserNotConnected
	}

	client := value.(*Client)

	item := jsonEncoderPool.Get().(*jsonEncoderPoolItem)
	item.buf.Reset()
	defer jsonEncoderPool.Put(item)

	if err := item.enc.Encode(message); err != nil {
		h.log.WithFields(ctx, logger.Fields{
			"user_id": userID,
			"action":  "ws_marshal",
			"type":    string(message.Type),
		}).Errorf("websocket marshal error: %v", err)
		return commonerrors.ErrMarshalError.WithCause(err)
	}

	messageBytes := item.buf.Bytes()
	if len(messageBytes) > 0 && messageBytes[len(messageBytes)-1] == '\n' {
		messageBytes = messageBytes[:len(messageBytes)-1]
	}

	messageBytesCopy := make([]byte, len(messageBytes))
	copy(messageBytesCopy, messageBytes)

	if err := h.sendWithTimeout(client.send, messageBytesCopy, userID, string(message.Type), ctx); err != nil {
		return err
	}

	h.log.WithFields(ctx, logger.Fields{
		"user_id": userID,
		"action":  "ws_send",
		"type":    string(message.Type),
	}).Info("message sent")
	return nil
}

func (h *Hub) SendErrorToUser(userID string, err error) {
	if err == nil {
		return
	}

	var domainErr commonerrors.DomainError
	if de, ok := commonerrors.AsDomainError(err); ok {
		domainErr = de
	} else {
		domainErr = commonerrors.ErrInternalError.WithCause(err)
	}

	errorPayload := ErrorPayload{
		Code:    domainErr.Code(),
		Message: domainErr.Message(),
	}
	payloadBytes, err := json.Marshal(errorPayload)
	if err != nil {
		h.log.WithFields(h.ctx, logger.Fields{
			"user_id": userID,
			"action":  "ws_error_marshal_failed",
		}).Warnf("websocket failed to marshal error payload: %v", err)
		return
	}
	errorMsg := &WSMessage{
		Type:    TypeError,
		Payload: payloadBytes,
	}
	h.SendToUser(userID, errorMsg)
}

func (h *Hub) sendWithTimeout(sendChan chan []byte, messageBytes []byte, userID, messageType string, ctx context.Context) error {
	sendCtx, cancel := context.WithTimeout(ctx, h.sendTimeout)
	defer cancel()
	select {
	case sendChan <- messageBytes:
		return nil
	case <-sendCtx.Done():
		select {
		case sendChan <- messageBytes:
			return nil
		default:
			observabilitymetrics.ChatWebSocketDroppedMessages.WithLabelValues(messageType).Inc()
			h.log.WithFields(ctx, logger.Fields{
				"user_id": userID,
				"action":  "ws_message_dropped",
				"type":    messageType,
			}).Warn("websocket message dropped due to slow client")
			return commonerrors.ErrClientTooSlow
		}
	}
}

func (h *Hub) IsUserOnline(userID string) bool {
	_, ok := h.clients.Load(userID)
	return ok
}

func (h *Hub) HandleMessage(client *Client, msg *WSMessage) {
	if h.messageHandler != nil {
		h.messageHandler.HandleMessage(client, msg)
	}
}

func (h *Hub) handleUnregister(client *Client) {
	if _, ok := h.clients.Load(client.userID); !ok {
		return
	}

	h.clients.Delete(client.userID)
	totalClients := h.clientCount.Add(-1)

	if h.fileService != nil {
		h.fileService.OnUserDisconnected(client.userID)
	}

	client.Stop()
	client.Close()
	client.WaitForShutdown(constants.WebSocketClientShutdownTimeout)
	observabilitymetrics.ChatWebSocketConnectionsActive.Dec()
	observabilitymetrics.ChatWebSocketDisconnections.WithLabelValues("unregister").Inc()
	h.log.WithFields(client.ctx, logger.Fields{
		"user_id":  client.userID,
		"username": client.username,
		"total":    totalClients,
		"action":   "ws_unregister",
	}).Info("websocket client unregistered")

	msg, err := marshalMessage(TypePeerDisconnected, PeerDisconnectedPayload{PeerID: client.userID})
	if err != nil {
		h.log.WithFields(client.ctx, logger.Fields{
			"user_id": client.userID,
			"action":  "ws_marshal_peer_disconnected",
		}).Errorf("websocket marshal peer_disconnected failed: %v", err)
		return
	}
	msgBytes, _ := json.Marshal(msg)
	h.clients.Range(func(key, value interface{}) bool {
		otherClient := value.(*Client)
		select {
		case otherClient.send <- msgBytes:
		default:
		}
		return true
	})
}

func (h *Hub) Shutdown() {
	h.cancel()
	if h.presenceService != nil {
		h.presenceService.Stop()
	}
	if h.messageHandler != nil {
		h.messageHandler.Shutdown()
	}
	h.shutdown()
}
