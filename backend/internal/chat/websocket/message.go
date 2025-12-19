package websocket

import "encoding/json"

type MessageType string

const (
	TypeAuth               MessageType = "auth"
	TypeEphemeralKey       MessageType = "ephemeral_key"
	TypeMessage            MessageType = "message"
	TypeSessionEstablished MessageType = "session_established"
	TypePeerOffline        MessageType = "peer_offline"
	TypePeerDisconnected   MessageType = "peer_disconnected"
	TypeFileStart          MessageType = "file_start"
	TypeFileChunk          MessageType = "file_chunk"
	TypeFileComplete       MessageType = "file_complete"
	TypeAck                MessageType = "ack"
)

type WSMessage struct {
	Type    MessageType     `json:"type"`
	Payload json.RawMessage `json:"payload"`
}

type EphemeralKeyPayload struct {
	To          string `json:"to"`
	From        string `json:"from,omitempty"`
	PublicKey   string `json:"public_key"`
	Signature   string `json:"signature"`
	MessageID   string `json:"message_id"`
	RequiresAck bool   `json:"requires_ack"`
}

type AckPayload struct {
	To        string `json:"to"`
	MessageID string `json:"message_id"`
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

type FileStartPayload struct {
	To          string `json:"to"`
	From        string `json:"from,omitempty"`
	FileID      string `json:"file_id"`
	Filename    string `json:"filename"`
	MimeType    string `json:"mime_type"`
	TotalSize   int64  `json:"total_size"`
	TotalChunks int    `json:"total_chunks"`
	ChunkSize   int    `json:"chunk_size"`
}

type FileChunkPayload struct {
	To          string `json:"to"`
	From        string `json:"from,omitempty"`
	FileID      string `json:"file_id"`
	ChunkIndex  int    `json:"chunk_index"`
	TotalChunks int    `json:"total_chunks"`
	Ciphertext  string `json:"ciphertext"`
	Nonce       string `json:"nonce"`
}

type FileCompletePayload struct {
	To     string `json:"to"`
	From   string `json:"from,omitempty"`
	FileID string `json:"file_id"`
}

type AuthPayload struct {
	Token string `json:"token"`
}
