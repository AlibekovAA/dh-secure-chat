export type MessageType =
  | 'ephemeral_key'
  | 'message'
  | 'session_established'
  | 'peer_offline'
  | 'peer_disconnected';

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

export type WSMessageWithPayload<T> = {
  type: MessageType;
  payload: T;
};
