package websocket

import (
	"encoding/json"
	"sync"

	"github.com/AlibekovAA/dh-secure-chat/backend/internal/common/logger"
)

type Hub struct {
	clients    map[string]*Client
	register   chan *Client
	unregister chan *Client
	mu         sync.RWMutex
	log        *logger.Logger
}

func NewHub(log *logger.Logger) *Hub {
	return &Hub{
		clients:    make(map[string]*Client),
		register:   make(chan *Client),
		unregister: make(chan *Client),
		log:        log,
	}
}

func (h *Hub) Register(client *Client) {
	h.register <- client
}

func (h *Hub) Unregister(client *Client) {
	h.unregister <- client
}

func (h *Hub) Run() {
	for {
		select {
		case client := <-h.register:
			h.mu.Lock()
			if existing, ok := h.clients[client.userID]; ok {
				h.log.Infof("websocket closing existing connection user_id=%s username=%s", existing.userID, existing.username)
				close(existing.send)
				delete(h.clients, existing.userID)
			}
			h.clients[client.userID] = client
			h.mu.Unlock()
			h.log.Infof("websocket client registered user_id=%s username=%s total=%d", client.userID, client.username, len(h.clients))

		case client := <-h.unregister:
			h.mu.Lock()
			if _, ok := h.clients[client.userID]; ok {
				delete(h.clients, client.userID)
				close(client.send)
				h.log.Infof("websocket client unregistered user_id=%s username=%s total=%d", client.userID, client.username, len(h.clients))

				disconnectedMsg := &WSMessage{
					Type:    TypePeerDisconnected,
					Payload: mustMarshal(PeerDisconnectedPayload{PeerID: client.userID}),
				}

				for _, otherClient := range h.clients {
					select {
					case otherClient.send <- mustMarshalJSON(disconnectedMsg):
					default:
					}
				}
			}
			h.mu.Unlock()
		}
	}
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

func (h *Hub) HandleMessage(client *Client, msg *WSMessage) {
	switch msg.Type {
	case TypeEphemeralKey:
		var payload EphemeralKeyPayload
		if err := json.Unmarshal(msg.Payload, &payload); err != nil {
			h.log.Warnf("websocket invalid ephemeral_key payload user_id=%s: %v", client.userID, err)
			return
		}

		if payload.To == "" {
			h.log.Warnf("websocket ephemeral_key missing 'to' field user_id=%s", client.userID)
			return
		}

		if payload.To == client.userID {
			h.log.Warnf("websocket ephemeral_key to self user_id=%s", client.userID)
			return
		}

		if !h.IsUserOnline(payload.To) {
			offlineMsg := &WSMessage{
				Type:    TypePeerOffline,
				Payload: mustMarshal(PeerOfflinePayload{PeerID: payload.To}),
			}
			h.SendToUser(client.userID, offlineMsg)
			h.log.Infof("websocket ephemeral_key to offline user from=%s to=%s", client.userID, payload.To)
			return
		}

		forwardMsg := &WSMessage{
			Type:    TypeEphemeralKey,
			Payload: msg.Payload,
		}
		if h.SendToUser(payload.To, forwardMsg) {
			h.log.Debugf("websocket ephemeral_key forwarded from=%s to=%s", client.userID, payload.To)
		}

	case TypeMessage:
		var payload MessagePayload
		if err := json.Unmarshal(msg.Payload, &payload); err != nil {
			h.log.Warnf("websocket invalid message payload user_id=%s: %v", client.userID, err)
			return
		}

		if payload.To == "" {
			h.log.Warnf("websocket message missing 'to' field user_id=%s", client.userID)
			return
		}

		if payload.To == client.userID {
			h.log.Warnf("websocket message to self user_id=%s", client.userID)
			return
		}

		if !h.IsUserOnline(payload.To) {
			offlineMsg := &WSMessage{
				Type:    TypePeerOffline,
				Payload: mustMarshal(PeerOfflinePayload{PeerID: payload.To}),
			}
			h.SendToUser(client.userID, offlineMsg)
			h.log.Infof("websocket message to offline user from=%s to=%s", client.userID, payload.To)
			return
		}

		forwardMsg := &WSMessage{
			Type:    TypeMessage,
			Payload: msg.Payload,
		}
		if h.SendToUser(payload.To, forwardMsg) {
			h.log.Debugf("websocket message forwarded from=%s to=%s", client.userID, payload.To)
		}

	case TypeSessionEstablished:
		var payload SessionEstablishedPayload
		if err := json.Unmarshal(msg.Payload, &payload); err != nil {
			h.log.Warnf("websocket invalid session_established payload user_id=%s: %v", client.userID, err)
			return
		}

		if payload.To == "" {
			h.log.Warnf("websocket session_established missing 'to' field user_id=%s", client.userID)
			return
		}

		if payload.To == client.userID {
			h.log.Warnf("websocket session_established to self user_id=%s", client.userID)
			return
		}

		if !h.IsUserOnline(payload.To) {
			h.log.Infof("websocket session_established to offline user from=%s to=%s", client.userID, payload.To)
			return
		}

		forwardMsg := &WSMessage{
			Type:    TypeSessionEstablished,
			Payload: msg.Payload,
		}
		if h.SendToUser(payload.To, forwardMsg) {
			h.log.Debugf("websocket session_established forwarded from=%s to=%s", client.userID, payload.To)
		}

	default:
		h.log.Warnf("websocket unknown message type user_id=%s type=%s", client.userID, msg.Type)
	}
}

func mustMarshal(v interface{}) json.RawMessage {
	data, err := json.Marshal(v)
	if err != nil {
		return json.RawMessage("{}")
	}
	return data
}

func mustMarshalJSON(v interface{}) []byte {
	data, err := json.Marshal(v)
	if err != nil {
		return []byte("{}")
	}
	return data
}
