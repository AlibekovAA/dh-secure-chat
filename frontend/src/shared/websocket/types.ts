export type MessageType =
  | 'auth'
  | 'ephemeral_key'
  | 'message'
  | 'session_established'
  | 'peer_offline'
  | 'peer_disconnected'
  | 'file_start'
  | 'file_chunk'
  | 'file_complete'
  | 'ack'
  | 'reaction'
  | 'typing'
  | 'read_receipt';

export type ConnectionState =
  | 'connecting'
  | 'connected'
  | 'disconnected'
  | 'error';

export type WSMessage = {
  type: MessageType;
  payload: unknown;
  sequence?: number;
};

export type EphemeralKeyPayload = {
  to: string;
  from?: string;
  public_key: string;
  signature: string;
  message_id: string;
  requires_ack: true;
};

export type AckPayload = {
  to: string;
  message_id: string;
};

export type MessagePayload = {
  to: string;
  ciphertext: string;
  nonce: string;
};

export type SessionEstablishedPayload = {
  to: string;
  peer_id: string;
};

export type PeerOfflinePayload = {
  peer_id: string;
};

export type PeerDisconnectedPayload = {
  peer_id: string;
};

export type FileStartPayload = {
  to: string;
  from?: string;
  file_id: string;
  filename: string;
  mime_type: string;
  total_size: number;
  total_chunks: number;
  chunk_size: number;
};

export type FileChunkPayload = {
  to: string;
  from?: string;
  file_id: string;
  chunk_index: number;
  total_chunks: number;
  ciphertext: string;
  nonce: string;
};

export type FileCompletePayload = {
  to: string;
  from?: string;
  file_id: string;
};

export type WSMessageWithPayload<T> = {
  type: MessageType;
  payload: T;
};

export type ReactionPayload = {
  to: string;
  from?: string;
  message_id: string;
  emoji: string;
  action: 'add' | 'remove';
};

export type TypingPayload = {
  to: string;
  from?: string;
  is_typing: boolean;
};

export type ReadReceiptPayload = {
  to: string;
  from?: string;
  message_id: string;
  read_at: number; // Unix timestamp
};
