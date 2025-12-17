package websocket

import "encoding/json"

type MessageType string

const (
	TypeEphemeralKey       MessageType = "ephemeral_key"
	TypeMessage            MessageType = "message"
	TypeSessionEstablished MessageType = "session_established"
	TypePeerOffline        MessageType = "peer_offline"
	TypePeerDisconnected   MessageType = "peer_disconnected"
)

type WSMessage struct {
	Type    MessageType     `json:"type"`
	Payload json.RawMessage `json:"payload"`
}

type EphemeralKeyPayload struct {
	To        string `json:"to"`
	PublicKey string `json:"public_key"`
}

type MessagePayload struct {
	To         string `json:"to"`
	Ciphertext string `json:"ciphertext"`
	Nonce      string `json:"nonce"`
}

type SessionEstablishedPayload struct {
	To     string `json:"to"`
	PeerID string `json:"peer_id"`
}

type PeerOfflinePayload struct {
	PeerID string `json:"peer_id"`
}

type PeerDisconnectedPayload struct {
	PeerID string `json:"peer_id"`
}
