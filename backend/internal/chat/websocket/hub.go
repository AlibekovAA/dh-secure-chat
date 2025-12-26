package websocket

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"regexp"
	"strconv"
	"sync"
	"sync/atomic"
	"time"

	"github.com/AlibekovAA/dh-secure-chat/backend/internal/chat/metrics"
	"github.com/AlibekovAA/dh-secure-chat/backend/internal/chat/transfer"
	"github.com/AlibekovAA/dh-secure-chat/backend/internal/chat/websocket/middleware"
	"github.com/AlibekovAA/dh-secure-chat/backend/internal/common/clock"
	commonerrors "github.com/AlibekovAA/dh-secure-chat/backend/internal/common/errors"
	"github.com/AlibekovAA/dh-secure-chat/backend/internal/common/logger"
	"github.com/AlibekovAA/dh-secure-chat/backend/internal/common/resilience"
	observabilitymetrics "github.com/AlibekovAA/dh-secure-chat/backend/internal/observability/metrics"
	userdomain "github.com/AlibekovAA/dh-secure-chat/backend/internal/user/domain"
	userrepo "github.com/AlibekovAA/dh-secure-chat/backend/internal/user/repository"
)

var audioMimePattern = regexp.MustCompile(`^(audio/(webm|ogg|mpeg|mp4|wav|x-m4a))`)

const userExistenceCacheTTL = 5 * time.Minute
const userExistenceCacheCleanupInterval = 1 * time.Minute

type userExistenceCacheEntry struct {
	exists    bool
	expiresAt time.Time
}

var jsonBufferPool = sync.Pool{
	New: func() interface{} {
		return &bytes.Buffer{}
	},
}

func isValidAudioMimeType(mimeType string) bool {
	return audioMimePattern.MatchString(mimeType)
}

type Hub struct {
	clients               sync.Map
	register              chan *Client
	unregister            chan *Client
	clientCount           atomic.Int64
	maxConnections        int
	log                   *logger.Logger
	userRepo              userrepo.Repository
	lastSeenUpdater       *LastSeenUpdater
	processor             *MessageProcessor
	fileTracker           transfer.Tracker
	idempotency           *IdempotencyTracker
	idempotencyMiddleware *middleware.IdempotencyMiddleware
	sendTimeout           time.Duration
	userExistenceCache    sync.Map
	clock                 clock.Clock
	debugSampleRate       float64
	ctx                   context.Context
	cancel                context.CancelFunc
}

type HubConfig struct {
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
	SendTimeout             time.Duration
	ShardCount              int
	MaxConnections          int
	DebugSampleRate         float64
}

func NewHub(log *logger.Logger, userRepo userrepo.Repository, config HubConfig) *Hub {
	ctx, cancel := context.WithCancel(context.Background())

	clk := clock.NewRealClock()
	hub := &Hub{
		register:        make(chan *Client),
		unregister:      make(chan *Client),
		log:             log,
		userRepo:        userRepo,
		fileTracker:     transfer.NewTracker(config.FileTransferTimeout, clk),
		idempotency:     NewIdempotencyTracker(ctx, config.IdempotencyTTL, clk),
		sendTimeout:     config.SendTimeout,
		maxConnections:  config.MaxConnections,
		clock:           clk,
		debugSampleRate: config.DebugSampleRate,
		ctx:             ctx,
		cancel:          cancel,
	}

	validator := NewDefaultValidator(config.MaxFileSize, config.MaxVoiceSize)
	router := NewMessageRouter(hub, validator, log)

	hub.processor = NewMessageProcessor(config.ProcessorWorkers, router, log, config.ProcessorQueueSize)

	idempotencyAdapter := &idempotencyAdapter{tracker: hub.idempotency}
	hub.idempotencyMiddleware = middleware.NewIdempotencyMiddleware(idempotencyAdapter, log)

	lastSeenCircuitBreaker := resilience.NewCircuitBreaker(resilience.CircuitBreakerConfig{
		Threshold:  config.CircuitBreakerThreshold,
		Timeout:    config.CircuitBreakerTimeout,
		ResetAfter: config.CircuitBreakerReset,
		Name:       "last_seen_update",
		Logger:     log,
	})
	hub.lastSeenUpdater = NewLastSeenUpdater(ctx, userRepo, log, config.LastSeenUpdateInterval, lastSeenCircuitBreaker, clk)

	go hub.fileTrackerCleanup()
	go hub.userExistenceCacheCleanup()

	return hub
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
				h.clients.Delete(client.userID)
				metrics.DecrementActiveWebSocketConnections()
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
				client.conn.Close()
				continue
			}

			h.clients.Store(client.userID, client)
			totalClients := h.clientCount.Add(1)
			metrics.IncrementActiveWebSocketConnections()
			h.log.WithFields(client.ctx, logger.Fields{
				"user_id":  client.userID,
				"username": client.username,
				"total":    totalClients,
				"action":   "ws_register",
			}).Info("websocket client registered")
			h.updateLastSeenDebounced(client.userID)

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
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
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
		duration := time.Since(startTime).Seconds()
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

	buf := jsonBufferPool.Get().(*bytes.Buffer)
	buf.Reset()
	defer jsonBufferPool.Put(buf)

	encoder := json.NewEncoder(buf)
	if err := encoder.Encode(message); err != nil {
		h.log.WithFields(ctx, logger.Fields{
			"user_id": userID,
			"action":  "ws_marshal",
			"type":    string(message.Type),
		}).Errorf("websocket marshal error: %v", err)
		return commonerrors.ErrMarshalError.WithCause(err)
	}

	messageBytes := buf.Bytes()
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

func (h *Hub) sendErrorToUser(userID string, err error) {
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
			metrics.IncrementDroppedMessages(messageType)
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

func (h *Hub) forwardMessage(ctx context.Context, msg *WSMessage, payload payloadWithTo, requireOnline bool, fromUserID string) bool {
	to := payload.GetTo()
	if to == "" {
		if fromUserID != "" {
			h.log.WithFields(ctx, logger.Fields{
				"user_id": fromUserID,
				"type":    string(msg.Type),
				"action":  "ws_message_missing_to",
			}).Warn("websocket message missing 'to' field")
		}
		return false
	}

	if fromUserID != "" && to == fromUserID {
		h.log.WithFields(ctx, logger.Fields{
			"user_id": fromUserID,
			"type":    string(msg.Type),
			"action":  "ws_message_to_self",
		}).Warn("websocket message to self")
		return false
	}

	if requireOnline && !h.IsUserOnline(to) {
		if fromUserID != "" {
			if err := h.sendPeerOffline(ctx, fromUserID, to); err != nil {
				h.log.WithFields(ctx, logger.Fields{
					"from":   fromUserID,
					"to":     to,
					"action": "ws_peer_offline_send",
				}).Errorf("websocket failed to send peer_offline: %v", err)
			}
		}
		h.log.WithFields(ctx, logger.Fields{
			"from":   fromUserID,
			"to":     to,
			"type":   string(msg.Type),
			"action": "ws_message_offline",
		}).Info("websocket message to offline user")
		return false
	}

	if !requireOnline {
		exists, err := h.checkUserExists(ctx, to)
		if err != nil {
			h.log.WithFields(ctx, logger.Fields{
				"from":   fromUserID,
				"to":     to,
				"type":   string(msg.Type),
				"action": "ws_user_check_failed",
			}).Errorf("websocket failed to check user existence: %v", err)
			return false
		}
		if !exists {
			if fromUserID != "" {
				h.log.WithFields(ctx, logger.Fields{
					"from":   fromUserID,
					"to":     to,
					"type":   string(msg.Type),
					"action": "ws_message_user_not_found",
				}).Warn("websocket message to non-existent user")
			}
			return false
		}
	}

	if err := h.SendToUserWithContext(ctx, to, msg); err != nil {
		if !errors.Is(err, context.Canceled) && !errors.Is(err, context.DeadlineExceeded) {
			h.log.WithFields(ctx, logger.Fields{
				"from":   fromUserID,
				"to":     to,
				"type":   string(msg.Type),
				"action": "ws_forward_failed",
			}).Warnf("websocket failed to forward message: %v", err)
		}
		return false
	}

	h.log.WithFields(ctx, logger.Fields{
		"from":   fromUserID,
		"to":     to,
		"type":   string(msg.Type),
		"action": "ws_message_forwarded",
	}).DebugSampled(h.debugSampleRate, "websocket message forwarded")
	return true
}

func (h *Hub) HandleMessage(client *Client, msg *WSMessage) {
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

func (h *Hub) handleUnregister(client *Client) {
	if _, ok := h.clients.Load(client.userID); !ok {
		return
	}

	h.clients.Delete(client.userID)
	totalClients := h.clientCount.Add(-1)

	transfers := h.fileTracker.GetTransfersForUser(client.userID)
	for _, tr := range transfers {
		h.notifyFileTransferFailed(tr)
		if err := h.fileTracker.Complete(tr.FileID); err != nil {
			h.log.WithFields(client.ctx, logger.Fields{
				"user_id": client.userID,
				"file_id": tr.FileID,
				"action":  "ws_file_complete_on_unregister",
			}).Warnf("websocket failed to complete file transfer on unregister: %v", err)
		}
	}

	client.Stop()
	client.Close()
	metrics.DecrementActiveWebSocketConnections()
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

func (h *Hub) sendPeerOffline(ctx context.Context, fromUserID, peerID string) error {
	msg, err := marshalMessage(TypePeerOffline, PeerOfflinePayload{PeerID: peerID})
	if err != nil {
		return commonerrors.ErrMarshalError.WithCause(err)
	}
	if err := h.SendToUserWithContext(ctx, fromUserID, msg); err != nil {
		return err
	}
	return nil
}

func (h *Hub) updateLastSeenDebounced(userID string) {
	if h.lastSeenUpdater != nil {
		h.lastSeenUpdater.Enqueue(userID)
	}
}

func (h *Hub) trackFileTransfer(payload FileStartPayload) {
	req := transfer.TrackRequest{
		FileID:      payload.FileID,
		From:        payload.From,
		To:          payload.To,
		TotalChunks: payload.TotalChunks,
	}

	if err := h.fileTracker.Track(req); err != nil {
		h.log.WithFields(h.ctx, logger.Fields{
			"file_id": payload.FileID,
			"from":    payload.From,
			"to":      payload.To,
			"action":  "ws_file_track",
		}).Warnf("websocket failed to track file transfer: %v", err)
		observabilitymetrics.ChatWebSocketFileTransferFailures.WithLabelValues("track_failed").Inc()
	}
}

func (h *Hub) updateFileTransferProgress(fileID string, chunkIndex int) {
	if err := h.fileTracker.UpdateProgress(fileID, chunkIndex); err != nil {
		if errors.Is(err, commonerrors.ErrTransferNotFound) {
			h.log.WithFields(h.ctx, logger.Fields{
				"file_id":     fileID,
				"chunk_index": chunkIndex,
				"action":      "ws_file_progress_skipped",
			}).Debug("websocket file transfer progress skipped (no tracking)")
			return
		}
		h.log.WithFields(h.ctx, logger.Fields{
			"file_id":     fileID,
			"chunk_index": chunkIndex,
			"action":      "ws_file_progress_failed",
		}).Warnf("websocket file transfer progress failed: %v", err)
	}
}

func (h *Hub) completeFileTransfer(fileID string) {
	transfers := h.fileTracker.GetTransfersForUser("")
	var tr *transfer.Transfer
	for _, t := range transfers {
		if t.FileID == fileID {
			tr = t
			break
		}
	}

	if err := h.fileTracker.Complete(fileID); err != nil {
		if errors.Is(err, commonerrors.ErrTransferNotFound) {
			h.log.WithFields(h.ctx, logger.Fields{
				"file_id": fileID,
				"action":  "ws_file_complete_skipped",
			}).Debug("websocket file transfer complete skipped (no tracking)")
			return
		}
		h.log.WithFields(h.ctx, logger.Fields{
			"file_id": fileID,
			"action":  "ws_file_complete_failed",
		}).Warnf("websocket failed to complete file transfer: %v", err)
		observabilitymetrics.ChatWebSocketFileTransferFailures.WithLabelValues("complete_failed").Inc()
		return
	}

	if tr != nil {
		duration := time.Since(tr.StartedAt).Seconds()
		observabilitymetrics.ChatWebSocketFileTransferDurationSeconds.WithLabelValues("success").Observe(duration)
	}
}

func (h *Hub) notifyFileTransferFailed(tr *transfer.Transfer) {
	if tr.To == "" {
		return
	}

	duration := time.Since(tr.StartedAt).Seconds()
	observabilitymetrics.ChatWebSocketFileTransferDurationSeconds.WithLabelValues("failed").Observe(duration)
	observabilitymetrics.ChatWebSocketFileTransferFailures.WithLabelValues("timeout_or_disconnect").Inc()

	msg, err := marshalMessage(TypeFileComplete, FileCompletePayload{
		To:     tr.To,
		From:   tr.From,
		FileID: tr.FileID,
	})
	if err != nil {
		h.log.WithFields(h.ctx, logger.Fields{
			"file_id": tr.FileID,
			"action":  "ws_file_failed_marshal",
		}).Errorf("websocket failed to marshal file_failed: %v", err)
		return
	}
	if err := h.SendToUserWithContext(h.ctx, tr.To, msg); err != nil {
		h.log.WithFields(h.ctx, logger.Fields{
			"to":      tr.To,
			"file_id": tr.FileID,
			"action":  "ws_file_failed_notify",
		}).Warnf("websocket failed to notify file transfer failure: %v", err)
	}
}

func (h *Hub) fileTrackerCleanup() {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-h.ctx.Done():
			return
		case <-ticker.C:
			if removed := h.fileTracker.CleanupStale(); removed > 0 {
				h.log.Debugf("websocket cleaned up stale file transfers count=%d", removed)
			}
		}
	}
}

func (h *Hub) checkUserExists(ctx context.Context, userID string) (bool, error) {
	if cached, ok := h.userExistenceCache.Load(userID); ok {
		entry := cached.(*userExistenceCacheEntry)
		if h.clock.Now().Before(entry.expiresAt) {
			observabilitymetrics.ChatWebSocketUserExistenceCacheHits.Inc()
			return entry.exists, nil
		}
		h.userExistenceCache.Delete(userID)
	}

	observabilitymetrics.ChatWebSocketUserExistenceCacheMisses.Inc()

	_, err := h.userRepo.FindByID(ctx, userdomain.ID(userID))
	exists := err == nil
	if err != nil && !errors.Is(err, userrepo.ErrUserNotFound) {
		return false, err
	}

	h.userExistenceCache.Store(userID, &userExistenceCacheEntry{
		exists:    exists,
		expiresAt: h.clock.Now().Add(userExistenceCacheTTL),
	})

	return exists, nil
}

func (h *Hub) userExistenceCacheCleanup() {
	ticker := time.NewTicker(userExistenceCacheCleanupInterval)
	defer ticker.Stop()

	for {
		select {
		case <-h.ctx.Done():
			return
		case <-ticker.C:
			now := h.clock.Now()
			removed := 0
			total := 0
			h.userExistenceCache.Range(func(key, value interface{}) bool {
				total++
				entry := value.(*userExistenceCacheEntry)
				if now.After(entry.expiresAt) {
					h.userExistenceCache.Delete(key)
					removed++
				}
				return true
			})
			observabilitymetrics.ChatWebSocketUserExistenceCacheSize.Set(float64(total))
			if removed > 0 {
				h.log.Debugf("websocket cleaned up stale user existence cache entries count=%d", removed)
			}
		}
	}
}

func (h *Hub) Shutdown() {
	h.cancel()
	if h.lastSeenUpdater != nil {
		h.lastSeenUpdater.Stop()
	}
	if h.idempotency != nil {
		h.idempotency.Shutdown()
	}
	h.processor.Shutdown()
	h.shutdown()
}

type idempotencyAdapter struct {
	tracker *IdempotencyTracker
}

func (a *idempotencyAdapter) GenerateOperationID(userID string, msgType string, payload []byte) string {
	return a.tracker.GenerateOperationID(userID, MessageType(msgType), payload)
}

func (a *idempotencyAdapter) Execute(operationID string, msgType string, fn func() (interface{}, error)) (interface{}, error) {
	return a.tracker.Execute(operationID, MessageType(msgType), fn)
}
