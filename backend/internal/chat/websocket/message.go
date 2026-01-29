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
	TypeTyping             MessageType = "typing"
	TypeReaction           MessageType = "reaction"
	TypeMessageDelete      MessageType = "message_delete"
	TypeMessageEdit        MessageType = "message_edit"
	TypeMessageRead        MessageType = "message_read"
	TypeError              MessageType = "error"
)

func (mt MessageType) String() string {
	return string(mt)
}

func (mt MessageType) IsValid() bool {
	switch mt {
	case TypeAuth, TypeEphemeralKey, TypeMessage, TypeSessionEstablished,
		TypePeerOffline, TypePeerDisconnected, TypeFileStart, TypeFileChunk,
		TypeFileComplete, TypeAck, TypeTyping, TypeReaction, TypeMessageDelete,
		TypeMessageEdit, TypeMessageRead, TypeError:
		return true
	default:
		return false
	}
}

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
	From       string `json:"from,omitempty"`
	MessageID  string `json:"message_id"`
	Ciphertext string `json:"ciphertext"`
	Nonce      string `json:"nonce"`
	ReplyToID  string `json:"reply_to_message_id,omitempty"`
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
	AccessMode  string `json:"access_mode,omitempty"`
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

type AuthResponsePayload struct {
	Authenticated bool   `json:"authenticated"`
	Code          string `json:"code,omitempty"`
	Message       string `json:"message,omitempty"`
}

type TypingPayload struct {
	To       string `json:"to"`
	From     string `json:"from,omitempty"`
	IsTyping bool   `json:"is_typing"`
}

type ReactionPayload struct {
	To        string `json:"to"`
	From      string `json:"from,omitempty"`
	MessageID string `json:"message_id"`
	Emoji     string `json:"emoji"`
	Action    string `json:"action"`
}

type MessageDeletePayload struct {
	To        string `json:"to"`
	From      string `json:"from,omitempty"`
	MessageID string `json:"message_id"`
	Scope     string `json:"scope,omitempty"`
}

type MessageEditPayload struct {
	To         string `json:"to"`
	From       string `json:"from,omitempty"`
	MessageID  string `json:"message_id"`
	Ciphertext string `json:"ciphertext"`
	Nonce      string `json:"nonce"`
}

type MessageReadPayload struct {
	To        string `json:"to"`
	From      string `json:"from,omitempty"`
	MessageID string `json:"message_id"`
}

type ErrorPayload struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}
