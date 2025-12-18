export type MessageType =
  | 'ephemeral_key'
  | 'message'
  | 'session_established'
  | 'peer_offline'
  | 'peer_disconnected'
  | 'file_start'
  | 'file_chunk'
  | 'file_complete';

export type ConnectionState =
  | 'connecting'
  | 'connected'
  | 'disconnected'
  | 'error';

export type WSMessage = {
  type: MessageType;
  payload: unknown;
};

export type EphemeralKeyPayload = {
  to: string;
  public_key: string;
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
  file_id: string;
  filename: string;
  mime_type: string;
  total_size: number;
  total_chunks: number;
  chunk_size: number;
};

export type FileChunkPayload = {
  to: string;
  file_id: string;
  chunk_index: number;
  total_chunks: number;
  ciphertext: string;
  nonce: string;
};

export type FileCompletePayload = {
  to: string;
  file_id: string;
};

export type WSMessageWithPayload<T> = {
  type: MessageType;
  payload: T;
};
