package websocket

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/AlibekovAA/dh-secure-chat/backend/internal/chat/metrics"
	"github.com/AlibekovAA/dh-secure-chat/backend/internal/common/logger"
	userdomain "github.com/AlibekovAA/dh-secure-chat/backend/internal/user/domain"
	userrepo "github.com/AlibekovAA/dh-secure-chat/backend/internal/user/repository"
)

const lastSeenUpdateInterval = 1 * time.Minute

type Hub struct {
	clients         map[string]*Client
	register        chan *Client
	unregister      chan *Client
	lastSeenUpdates map[string]time.Time
	mu              sync.RWMutex
	log             *logger.Logger
	userRepo        userrepo.Repository
}

func NewHub(log *logger.Logger, userRepo userrepo.Repository) *Hub {
	return &Hub{
		clients:         make(map[string]*Client),
		register:        make(chan *Client),
		unregister:      make(chan *Client),
		lastSeenUpdates: make(map[string]time.Time),
		log:             log,
		userRepo:        userRepo,
	}
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
			h.mu.Lock()
			if existing, ok := h.clients[client.userID]; ok {
				h.log.Infof("websocket closing existing connection user_id=%s username=%s", existing.userID, existing.username)
				close(existing.send)
				delete(h.clients, existing.userID)
				metrics.DecrementActiveWebSocketConnections()
			}
			h.clients[client.userID] = client
			totalClients := len(h.clients)
			h.mu.Unlock()
			metrics.IncrementActiveWebSocketConnections()
			h.log.Infof("websocket client registered user_id=%s username=%s total=%d", client.userID, client.username, totalClients)
			h.updateLastSeenDebounced(client.userID)

		case client := <-h.unregister:
			h.handleUnregister(client)
		}
	}
}

func (h *Hub) shutdown() {
	h.mu.Lock()
	defer h.mu.Unlock()

	for id, client := range h.clients {
		close(client.send)
		delete(h.clients, id)
	}

	h.log.Infof("websocket hub shutdown completed")
}

func (h *Hub) SendToUser(userID string, message *WSMessage) bool {
	h.mu.RLock()
	client, ok := h.clients[userID]
	h.mu.RUnlock()

	if !ok {
		return false
	}

	messageBytes, err := json.Marshal(message)
	if err != nil {
		h.log.Errorf("websocket marshal error: %v", err)
		return false
	}

	select {
	case client.send <- messageBytes:
		return true
	default:
		h.log.Warnf("websocket send buffer full user_id=%s", userID)
		return false
	}
}

func (h *Hub) IsUserOnline(userID string) bool {
	h.mu.RLock()
	_, ok := h.clients[userID]
	h.mu.RUnlock()
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

func (h *Hub) forwardMessage(client *Client, msg *WSMessage, payload payloadWithTo, requireOnline bool) bool {
	to := payload.GetTo()
	if to == "" {
		h.log.Warnf("websocket message missing 'to' field user_id=%s type=%s", client.userID, msg.Type)
		return false
	}

	if to == client.userID {
		h.log.Warnf("websocket message to self user_id=%s type=%s", client.userID, msg.Type)
		return false
	}

	if requireOnline && !h.IsUserOnline(to) {
		if err := h.sendPeerOffline(client.userID, to); err != nil {
			h.log.Errorf("websocket failed to send peer_offline from=%s to=%s: %v", client.userID, to, err)
		}
		h.log.Infof("websocket message to offline user from=%s to=%s type=%s", client.userID, to, msg.Type)
		return false
	}

	forwardMsg := &WSMessage{
		Type:    msg.Type,
		Payload: msg.Payload,
	}
	if h.SendToUser(to, forwardMsg) {
		h.log.Debugf("websocket message forwarded from=%s to=%s type=%s", client.userID, to, msg.Type)
		return true
	}
	return false
}

func (h *Hub) forwardMessageWithModifiedPayload(msg *WSMessage, payload payloadWithTo, requireOnline bool, fromUserID string) bool {
	to := payload.GetTo()
	if to == "" {
		return false
	}

	if requireOnline && !h.IsUserOnline(to) {
		if fromUserID != "" {
			if err := h.sendPeerOffline(fromUserID, to); err != nil {
				h.log.Errorf("websocket failed to send peer_offline from=%s to=%s: %v", fromUserID, to, err)
			}
		}
		h.log.Infof("websocket message to offline user from=%s to=%s type=%s", fromUserID, to, msg.Type)
		return false
	}

	if h.SendToUser(to, msg) {
		h.log.Debugf("websocket message forwarded from=%s to=%s type=%s", fromUserID, to, msg.Type)
		return true
	}
	return false
}

func (h *Hub) HandleMessage(client *Client, msg *WSMessage) {
	switch msg.Type {
	case TypeEphemeralKey:
		var payload EphemeralKeyPayload
		if err := json.Unmarshal(msg.Payload, &payload); err != nil {
			h.log.Warnf("websocket invalid ephemeral_key payload user_id=%s: %v", client.userID, err)
			metrics.IncrementWebSocketError("invalid_ephemeral_key_payload")
			return
		}
		payload.From = client.userID
		payloadBytes, err := json.Marshal(payload)
		if err != nil {
			h.log.Warnf("websocket failed to marshal ephemeral_key payload user_id=%s: %v", client.userID, err)
			return
		}
		forwardMsg := &WSMessage{
			Type:    msg.Type,
			Payload: payloadBytes,
		}
		h.forwardMessageWithModifiedPayload(forwardMsg, payload, true, client.userID)

	case TypeMessage:
		var payload MessagePayload
		if err := json.Unmarshal(msg.Payload, &payload); err != nil {
			h.log.Warnf("websocket invalid message payload user_id=%s: %v", client.userID, err)
			return
		}
		h.forwardMessage(client, msg, payload, true)

	case TypeSessionEstablished:
		var payload SessionEstablishedPayload
		if err := json.Unmarshal(msg.Payload, &payload); err != nil {
			h.log.Warnf("websocket invalid session_established payload user_id=%s: %v", client.userID, err)
			return
		}
		h.forwardMessage(client, msg, payload, false)

	case TypeFileStart:
		var payload FileStartPayload
		if err := json.Unmarshal(msg.Payload, &payload); err != nil {
			h.log.Warnf("websocket invalid file_start payload user_id=%s: %v", client.userID, err)
			return
		}
		h.forwardMessage(client, msg, payload, true)

	case TypeFileChunk:
		var payload FileChunkPayload
		if err := json.Unmarshal(msg.Payload, &payload); err != nil {
			h.log.Warnf("websocket invalid file_chunk payload user_id=%s: %v", client.userID, err)
			return
		}
		h.forwardMessage(client, msg, payload, true)

	case TypeFileComplete:
		var payload FileCompletePayload
		if err := json.Unmarshal(msg.Payload, &payload); err != nil {
			h.log.Warnf("websocket invalid file_complete payload user_id=%s: %v", client.userID, err)
			return
		}
		h.forwardMessage(client, msg, payload, true)

	case TypeAck:
		var payload AckPayload
		if err := json.Unmarshal(msg.Payload, &payload); err != nil {
			h.log.Warnf("websocket invalid ack payload user_id=%s: %v", client.userID, err)
			return
		}
		h.forwardMessage(client, msg, payload, false)

	default:
		h.log.Warnf("websocket unknown message type user_id=%s type=%s", client.userID, msg.Type)
		metrics.IncrementWebSocketError("unknown_message_type")
	}
}

func (h *Hub) handleUnregister(client *Client) {
	h.mu.Lock()
	defer h.mu.Unlock()

	if _, ok := h.clients[client.userID]; !ok {
		return
	}

	delete(h.clients, client.userID)
	totalClients := len(h.clients)
	close(client.send)
	metrics.DecrementActiveWebSocketConnections()
	h.log.Infof("websocket client unregistered user_id=%s username=%s total=%d", client.userID, client.username, totalClients)

	disconnectedMsg := &WSMessage{
		Type:    TypePeerDisconnected,
		Payload: json.RawMessage{},
	}

	payload, err := json.Marshal(PeerDisconnectedPayload{PeerID: client.userID})
	if err != nil {
		h.log.Errorf("websocket marshal peer_disconnected payload failed user_id=%s: %v", client.userID, err)
		return
	}
	disconnectedMsg.Payload = payload

	messageBytes, err := json.Marshal(disconnectedMsg)
	if err != nil {
		h.log.Errorf("websocket marshal peer_disconnected message failed user_id=%s: %v", client.userID, err)
		return
	}

	for _, otherClient := range h.clients {
		select {
		case otherClient.send <- messageBytes:
		default:
		}
	}
}

func (h *Hub) sendPeerOffline(fromUserID, peerID string) error {
	payload, err := json.Marshal(PeerOfflinePayload{PeerID: peerID})
	if err != nil {
		return err
	}

	msg := &WSMessage{
		Type:    TypePeerOffline,
		Payload: payload,
	}

	if ok := h.SendToUser(fromUserID, msg); !ok {
		return fmt.Errorf("user %s is not connected", fromUserID)
	}

	return nil
}

func (h *Hub) updateLastSeenDebounced(userID string) {
	h.mu.RLock()
	lastUpdate, exists := h.lastSeenUpdates[userID]
	h.mu.RUnlock()

	now := time.Now()
	if exists && now.Sub(lastUpdate) < lastSeenUpdateInterval {
		return
	}

	h.mu.Lock()
	h.lastSeenUpdates[userID] = now
	h.mu.Unlock()

	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		if err := h.userRepo.UpdateLastSeen(ctx, userdomain.ID(userID)); err != nil {
			h.log.Warnf("websocket failed to update last_seen user_id=%s: %v", userID, err)
		}
	}()
}
