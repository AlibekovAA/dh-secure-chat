package websocket

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/AlibekovAA/dh-secure-chat/backend/internal/chat/metrics"
	"github.com/AlibekovAA/dh-secure-chat/backend/internal/common/logger"
	userdomain "github.com/AlibekovAA/dh-secure-chat/backend/internal/user/domain"
	userrepo "github.com/AlibekovAA/dh-secure-chat/backend/internal/user/repository"
)

func isValidAudioMimeType(mimeType string) bool {
	validTypes := []string{
		"audio/webm",
		"audio/ogg",
		"audio/mpeg",
		"audio/mp4",
		"audio/wav",
		"audio/x-m4a",
	}
	for _, validType := range validTypes {
		if len(mimeType) >= len(validType) && mimeType[:len(validType)] == validType {
			return true
		}
	}
	return false
}

type Hub struct {
	clients                sync.Map
	register               chan *Client
	unregister             chan *Client
	lastSeenUpdates        sync.Map
	log                    *logger.Logger
	userRepo               userrepo.Repository
	lastSeenUpdateInterval time.Duration
	processor              *MessageProcessor
	fileTracker            *FileTransferTracker
	circuitBreaker         *CircuitBreaker
	idempotency            *IdempotencyTracker
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
}

func NewHub(log *logger.Logger, userRepo userrepo.Repository, config HubConfig) *Hub {
	ctx, cancel := context.WithCancel(context.Background())

	validator := NewDefaultValidator(config.MaxFileSize, config.MaxVoiceSize)
	router := NewMessageRouter(nil, validator, log)

	hub := &Hub{
		register:               make(chan *Client),
		unregister:             make(chan *Client),
		log:                    log,
		userRepo:               userRepo,
		lastSeenUpdateInterval: config.LastSeenUpdateInterval,
		fileTracker:            NewFileTransferTracker(config.FileTransferTimeout),
		circuitBreaker:         NewCircuitBreaker(config.CircuitBreakerThreshold, config.CircuitBreakerTimeout, config.CircuitBreakerReset),
		idempotency:            NewIdempotencyTracker(config.IdempotencyTTL),
		ctx:                    ctx,
		cancel:                 cancel,
	}

	router.hub = hub
	hub.processor = NewMessageProcessor(config.ProcessorWorkers, router, log, config.ProcessorQueueSize)

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
				h.log.Infof("websocket closing existing connection user_id=%s username=%s", existingClient.userID, existingClient.username)
				existingClient.Stop()
				close(existingClient.send)
				h.clients.Delete(client.userID)
				metrics.DecrementActiveWebSocketConnections()
			}
			h.clients.Store(client.userID, client)
			totalClients := h.countClients()
			metrics.IncrementActiveWebSocketConnections()
			h.log.Infof("websocket client registered user_id=%s username=%s total=%d", client.userID, client.username, totalClients)
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

	for _, client := range clients {
		client.Stop()
		select {
		case client.send <- []byte(`{"type":"shutdown"}`):
		default:
		}
		close(client.send)
	}

	h.clients.Range(func(key, value interface{}) bool {
		h.clients.Delete(key)
		return true
	})

	h.log.Infof("websocket hub shutdown completed clients=%d", len(clients))
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
		return fmt.Errorf("user %s not connected", userID)
	}

	client := value.(*Client)
	sendChan := client.send

	messageBytes, err := json.Marshal(message)
	if err != nil {
		h.log.Errorf("websocket marshal error: %v", err)
		return fmt.Errorf("marshal error: %w", err)
	}

	if h.sendWithRetry(sendChan, messageBytes, userID, string(message.Type)) {
		return nil
	}

	return fmt.Errorf("failed to send message to user %s", userID)
}

func (h *Hub) sendWithRetry(sendChan chan []byte, messageBytes []byte, userID, messageType string) bool {
	const maxRetries = 3
	const baseDelay = 10 * time.Millisecond
	const maxDelay = 100 * time.Millisecond

	select {
	case sendChan <- messageBytes:
		return true
	default:
		h.log.Warnf("websocket send buffer full user_id=%s type=%s, attempting retry", userID, messageType)
	}

	for attempt := 0; attempt < maxRetries; attempt++ {
		select {
		case <-sendChan:
			select {
			case sendChan <- messageBytes:
				return true
			default:
				delay := baseDelay * (1 << attempt)
				if delay > maxDelay {
					delay = maxDelay
				}
				time.Sleep(delay)
			}
		default:
			delay := baseDelay * (1 << attempt)
			if delay > maxDelay {
				delay = maxDelay
			}
			time.Sleep(delay)

			select {
			case sendChan <- messageBytes:
				return true
			default:
				continue
			}
		}
	}

	h.log.Warnf("websocket failed to send after %d retries user_id=%s type=%s", maxRetries, userID, messageType)
	return false
}

func (h *Hub) IsUserOnline(userID string) bool {
	_, ok := h.clients.Load(userID)
	return ok
}

func (h *Hub) countClients() int {
	count := 0
	h.clients.Range(func(key, value interface{}) bool {
		count++
		return true
	})
	return count
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

func (h *Hub) forwardMessage(ctx context.Context, msg *WSMessage, payload payloadWithTo, requireOnline bool, fromUserID string) bool {
	to := payload.GetTo()
	if to == "" {
		if fromUserID != "" {
			h.log.Warnf("websocket message missing 'to' field from=%s type=%s", fromUserID, msg.Type)
		}
		return false
	}

	if fromUserID != "" && to == fromUserID {
		h.log.Warnf("websocket message to self from=%s type=%s", fromUserID, msg.Type)
		return false
	}

	if requireOnline && !h.IsUserOnline(to) {
		if fromUserID != "" {
			if err := h.sendPeerOffline(ctx, fromUserID, to); err != nil {
				h.log.Errorf("websocket failed to send peer_offline from=%s to=%s: %v", fromUserID, to, err)
			}
		}
		h.log.Infof("websocket message to offline user from=%s to=%s type=%s", fromUserID, to, msg.Type)
		return false
	}

	if err := h.SendToUserWithContext(ctx, to, msg); err != nil {
		if !errors.Is(err, context.Canceled) && !errors.Is(err, context.DeadlineExceeded) {
			h.log.Warnf("websocket failed to forward message from=%s to=%s type=%s: %v", fromUserID, to, msg.Type, err)
		}
		return false
	}

	h.log.Debugf("websocket message forwarded from=%s to=%s type=%s", fromUserID, to, msg.Type)
	return true
}

func (h *Hub) HandleMessage(client *Client, msg *WSMessage) {
	if msg.Type == TypeEphemeralKey {
		operationID := h.idempotency.generateOperationID(client.userID, msg.Type, msg.Payload)
		_, err := h.idempotency.Execute(operationID, func() (interface{}, error) {
			h.processor.Submit(client.ctx, client, msg)
			return nil, nil
		})
		if err != nil {
			h.log.Warnf("websocket idempotency check failed user_id=%s: %v", client.userID, err)
		}
		return
	}

	h.processor.Submit(client.ctx, client, msg)
}

func (h *Hub) handleUnregister(client *Client) {
	if _, ok := h.clients.Load(client.userID); !ok {
		return
	}

	h.clients.Delete(client.userID)
	totalClients := h.countClients()
	clients := make([]*Client, 0)
	h.clients.Range(func(key, value interface{}) bool {
		clients = append(clients, value.(*Client))
		return true
	})

	transfers := h.fileTracker.GetTransfersForUser(client.userID)
	for _, transfer := range transfers {
		h.notifyFileTransferFailed(transfer)
		h.fileTracker.Complete(transfer.FileID)
	}

	client.Stop()
	close(client.send)
	metrics.DecrementActiveWebSocketConnections()
	h.log.Infof("websocket client unregistered user_id=%s username=%s total=%d", client.userID, client.username, totalClients)

	payloadBytes, err := json.Marshal(PeerDisconnectedPayload{PeerID: client.userID})
	if err != nil {
		h.log.Errorf("websocket marshal peer_disconnected payload failed user_id=%s: %v", client.userID, err)
		return
	}

	disconnectedMsg := &WSMessage{
		Type:    TypePeerDisconnected,
		Payload: payloadBytes,
	}

	messageBytes, err := json.Marshal(disconnectedMsg)
	if err != nil {
		h.log.Errorf("websocket marshal peer_disconnected message failed user_id=%s: %v", client.userID, err)
		return
	}

	for _, otherClient := range clients {
		select {
		case otherClient.send <- messageBytes:
		default:
		}
	}
}

func (h *Hub) sendPeerOffline(ctx context.Context, fromUserID, peerID string) error {
	payloadBytes, err := json.Marshal(PeerOfflinePayload{PeerID: peerID})
	if err != nil {
		return fmt.Errorf("marshal error: %w", err)
	}

	msg := &WSMessage{
		Type:    TypePeerOffline,
		Payload: payloadBytes,
	}

	if err := h.SendToUserWithContext(ctx, fromUserID, msg); err != nil {
		return fmt.Errorf("user %s is not connected: %w", fromUserID, err)
	}

	return nil
}

func (h *Hub) updateLastSeenDebounced(userID string) {
	now := time.Now()
	value, exists := h.lastSeenUpdates.Load(userID)
	if exists {
		lastUpdate := value.(time.Time)
		if now.Sub(lastUpdate) < h.lastSeenUpdateInterval {
			return
		}
	}
	h.lastSeenUpdates.Store(userID, now)

	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		err := h.circuitBreaker.Call(ctx, func(ctx context.Context) error {
			return h.userRepo.UpdateLastSeen(ctx, userdomain.ID(userID))
		})

		if err != nil && !errors.Is(err, ErrCircuitOpen) {
			h.log.Warnf("websocket failed to update last_seen user_id=%s: %v", userID, err)
		}
	}()
}

func (h *Hub) trackFileTransfer(payload FileStartPayload) {
	h.fileTracker.Track(payload)
}

func (h *Hub) updateFileTransferProgress(fileID string, chunkIndex int) {
	h.fileTracker.UpdateProgress(fileID, chunkIndex)
}

func (h *Hub) completeFileTransfer(fileID string) {
	h.fileTracker.Complete(fileID)
}

func (h *Hub) notifyFileTransferFailed(transfer *FileTransfer) {
	payloadBytes, err := json.Marshal(FileCompletePayload{
		To:     transfer.To,
		From:   transfer.From,
		FileID: transfer.FileID,
	})
	if err != nil {
		h.log.Errorf("websocket failed to marshal file_failed payload: %v", err)
		return
	}

	msg := &WSMessage{
		Type:    TypeFileComplete,
		Payload: payloadBytes,
	}

	if transfer.To != "" {
		if err := h.SendToUserWithContext(h.ctx, transfer.To, msg); err != nil {
			h.log.Warnf("websocket failed to notify file transfer failure to=%s file_id=%s: %v", transfer.To, transfer.FileID, err)
		}
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
			h.fileTracker.CleanupStale()
		}
	}
}

func (h *Hub) Shutdown() {
	h.cancel()
	h.processor.Shutdown()
	h.shutdown()
}
