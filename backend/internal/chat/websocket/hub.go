package websocket

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"sync"
	"sync/atomic"
	"time"

	"github.com/AlibekovAA/dh-secure-chat/backend/internal/chat/metrics"
	"github.com/AlibekovAA/dh-secure-chat/backend/internal/chat/transfer"
	commonerrors "github.com/AlibekovAA/dh-secure-chat/backend/internal/common/errors"
	"github.com/AlibekovAA/dh-secure-chat/backend/internal/common/logger"
	prommetrics "github.com/AlibekovAA/dh-secure-chat/backend/internal/common/prometheus"
	userrepo "github.com/AlibekovAA/dh-secure-chat/backend/internal/user/repository"
)

var audioMimePattern = regexp.MustCompile(`^(audio/(webm|ogg|mpeg|mp4|wav|x-m4a))`)

const debugSampleRate = 0.01

func isValidAudioMimeType(mimeType string) bool {
	return audioMimePattern.MatchString(mimeType)
}

type Hub struct {
	clients                sync.Map
	register               chan *Client
	unregister             chan *Client
	clientCount            atomic.Int64
	log                    *logger.Logger
	userRepo               userrepo.Repository
	lastSeenUpdateInterval time.Duration
	lastSeenUpdater        *LastSeenUpdater
	processor              *MessageProcessor
	fileTracker            transfer.Tracker
	circuitBreaker         *CircuitBreaker
	idempotency            *IdempotencyTracker
	sendTimeout            time.Duration
	ctx                    context.Context
	cancel                 context.CancelFunc
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
}

func NewHub(log *logger.Logger, userRepo userrepo.Repository, config HubConfig) *Hub {
	ctx, cancel := context.WithCancel(context.Background())

	hub := &Hub{
		register:               make(chan *Client),
		unregister:             make(chan *Client),
		log:                    log,
		userRepo:               userRepo,
		lastSeenUpdateInterval: config.LastSeenUpdateInterval,
		fileTracker:            transfer.NewTracker(config.FileTransferTimeout),
		circuitBreaker:         NewCircuitBreaker(config.CircuitBreakerThreshold, config.CircuitBreakerTimeout, config.CircuitBreakerReset),
		idempotency:            NewIdempotencyTracker(config.IdempotencyTTL),
		sendTimeout:            config.SendTimeout,
		ctx:                    ctx,
		cancel:                 cancel,
	}

	validator := NewDefaultValidator(config.MaxFileSize, config.MaxVoiceSize)
	router := NewMessageRouter(hub, validator, log)

	hub.processor = NewMessageProcessor(config.ProcessorWorkers, router, log, config.ProcessorQueueSize)

	hub.lastSeenUpdater = NewLastSeenUpdater(ctx, userRepo, log, config.LastSeenUpdateInterval, hub.circuitBreaker)

	go hub.fileTrackerCleanup()

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
				close(existingClient.send)
				h.clients.Delete(client.userID)
				metrics.DecrementActiveWebSocketConnections()
				h.clientCount.Add(-1)
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
		close(client.send)
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
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	value, ok := h.clients.Load(userID)
	if !ok {
		return fmt.Errorf("user %s not connected: %w", userID, commonerrors.ErrUserNotConnected)
	}

	client := value.(*Client)
	messageBytes, err := json.Marshal(message)
	if err != nil {
		h.log.WithFields(ctx, logger.Fields{
			"user_id": userID,
			"action":  "ws_marshal",
			"type":    string(message.Type),
		}).Errorf("websocket marshal error: %v", err)
		return fmt.Errorf("marshal error: %w", err)
	}
	if err := h.sendWithTimeout(client.send, messageBytes, userID, string(message.Type), ctx); err != nil {
		return err
	}

	h.log.WithFields(ctx, logger.Fields{
		"user_id": userID,
		"action":  "ws_send",
		"type":    string(message.Type),
	}).Info("message sent")
	return nil
}

func (h *Hub) sendWithTimeout(sendChan chan []byte, messageBytes []byte, userID, messageType string, ctx context.Context) error {
	sendCtx, cancel := context.WithTimeout(ctx, h.sendTimeout)
	defer cancel()
	select {
	case sendChan <- messageBytes:
		return nil
	case <-sendCtx.Done():
		h.log.WithFields(ctx, logger.Fields{
			"user_id": userID,
			"action":  "ws_send_timeout",
			"type":    messageType,
		}).Warn("websocket send timed out")
		return fmt.Errorf("send timeout: %w", sendCtx.Err())
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
	}).DebugSampled(debugSampleRate, "websocket message forwarded")
	return true
}

func (h *Hub) HandleMessage(client *Client, msg *WSMessage) {
	switch msg.Type {
	case TypeEphemeralKey:
		operationID := h.idempotency.generateOperationID(client.userID, msg.Type, msg.Payload)
		_, err := h.idempotency.Execute(operationID, msg.Type, func() (interface{}, error) {
			h.processor.Submit(client.ctx, client, msg)
			return nil, nil
		})
		if err != nil {
			h.log.WithFields(client.ctx, logger.Fields{
				"user_id": client.userID,
				"type":    string(msg.Type),
				"action":  "ws_idempotency_failed",
			}).Warnf("websocket idempotency check failed: %v", err)
		}
		return

	case TypeMessage:
		var payload MessagePayload
		if err := json.Unmarshal(msg.Payload, &payload); err == nil && payload.MessageID != "" {
			operationID := h.idempotency.generateOperationID(client.userID+":"+payload.MessageID, msg.Type, msg.Payload)
			_, err := h.idempotency.Execute(operationID, msg.Type, func() (interface{}, error) {
				h.processor.Submit(client.ctx, client, msg)
				return nil, nil
			})
			if err != nil {
				h.log.WithFields(client.ctx, logger.Fields{
					"user_id":    client.userID,
					"message_id": payload.MessageID,
					"type":       string(msg.Type),
					"action":     "ws_idempotency_failed",
				}).Warnf("websocket idempotency check failed: %v", err)
			}
			return
		}

	case TypeFileChunk:
		var payload FileChunkPayload
		if err := json.Unmarshal(msg.Payload, &payload); err == nil && payload.FileID != "" {
			chunkKey := fmt.Sprintf("%s:%s:%d", client.userID, payload.FileID, payload.ChunkIndex)
			operationID := h.idempotency.generateOperationID(chunkKey, msg.Type, msg.Payload)
			_, err := h.idempotency.Execute(operationID, msg.Type, func() (interface{}, error) {
				h.processor.Submit(client.ctx, client, msg)
				return nil, nil
			})
			if err != nil {
				h.log.WithFields(client.ctx, logger.Fields{
					"user_id":     client.userID,
					"file_id":     payload.FileID,
					"chunk_index": payload.ChunkIndex,
					"type":        string(msg.Type),
					"action":      "ws_idempotency_failed",
				}).Warnf("websocket idempotency check failed: %v", err)
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
	close(client.send)
	metrics.DecrementActiveWebSocketConnections()
	prommetrics.ChatWebSocketDisconnections.WithLabelValues("unregister").Inc()
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
		return fmt.Errorf("marshal error: %w", err)
	}
	if err := h.SendToUserWithContext(ctx, fromUserID, msg); err != nil {
		return fmt.Errorf("user %s is not connected: %w", fromUserID, err)
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
		prommetrics.ChatWebSocketFileTransferFailures.WithLabelValues("track_failed").Inc()
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
		prommetrics.ChatWebSocketFileTransferFailures.WithLabelValues("complete_failed").Inc()
		return
	}

	if tr != nil {
		duration := time.Since(tr.StartedAt).Seconds()
		prommetrics.ChatWebSocketFileTransferDurationSeconds.WithLabelValues("success").Observe(duration)
	}
}

func (h *Hub) notifyFileTransferFailed(tr *transfer.Transfer) {
	if tr.To == "" {
		return
	}

	duration := time.Since(tr.StartedAt).Seconds()
	prommetrics.ChatWebSocketFileTransferDurationSeconds.WithLabelValues("failed").Observe(duration)
	prommetrics.ChatWebSocketFileTransferFailures.WithLabelValues("timeout_or_disconnect").Inc()

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

func (h *Hub) Shutdown() {
	h.cancel()
	if h.lastSeenUpdater != nil {
		h.lastSeenUpdater.Stop()
	}
	h.processor.Shutdown()
	h.shutdown()
}
