import { useCallback } from 'react';
import type {
  WSMessage,
  EphemeralKeyPayload,
  MessagePayload,
  SessionEstablishedPayload,
  PeerOfflinePayload,
  PeerDisconnectedPayload,
  FileStartPayload,
  FileChunkPayload,
  FileCompletePayload,
  AckPayload,
  TypingPayload,
  ReactionPayload,
  MessageDeletePayload,
  MessageEditPayload,
  MessageReadPayload,
} from '@/shared/websocket/types';
import type {
  ChatMessage,
  ChatSessionState,
  FileBuffer,
} from '@/modules/chat/useChatSession';
import {
  AES_GCM_NONCE_SIZE_BYTES,
  AES_GCM_TAG_SIZE_BYTES,
  FILE_CHUNK_SIZE,
  MAX_CONCURRENT_FILE_TRANSFERS,
  MAX_FILE_CHUNKS,
  MAX_FILE_SIZE,
  MAX_VOICE_SIZE,
} from '@/shared/constants';
import { MESSAGES } from '@/shared/messages';

const BASE64_PATTERN =
  /^(?:[A-Za-z0-9+/]{4})*(?:[A-Za-z0-9+/]{2}==|[A-Za-z0-9+/]{3}=)?$/;

function decodedBase64Length(
  value: unknown,
  maxEncodedLength: number
): number | null {
  if (
    typeof value !== 'string' ||
    value.length === 0 ||
    value.length > maxEncodedLength ||
    value.length % 4 !== 0 ||
    !BASE64_PATTERN.test(value)
  ) {
    return null;
  }

  try {
    const decoded = atob(value);
    return btoa(decoded) === value ? decoded.length : null;
  } catch {
    return null;
  }
}

function isValidFileMetadata(payload: FileStartPayload): boolean {
  if (
    typeof payload.file_id !== 'string' ||
    typeof payload.from !== 'string' ||
    typeof payload.to !== 'string' ||
    typeof payload.mime_type !== 'string'
  ) {
    return false;
  }

  const maxSize = payload.mime_type.startsWith('audio/')
    ? MAX_VOICE_SIZE
    : MAX_FILE_SIZE;

  return (
    payload.file_id.length > 0 &&
    payload.from.length > 0 &&
    payload.to.length > 0 &&
    Number.isSafeInteger(payload.total_size) &&
    payload.total_size > 0 &&
    payload.total_size <= maxSize &&
    Number.isSafeInteger(payload.chunk_size) &&
    payload.chunk_size > 0 &&
    payload.chunk_size <= FILE_CHUNK_SIZE &&
    Number.isSafeInteger(payload.total_chunks) &&
    payload.total_chunks > 0 &&
    payload.total_chunks <= MAX_FILE_CHUNKS &&
    payload.total_chunks === Math.ceil(payload.total_size / payload.chunk_size)
  );
}

function expectedChunkSize(
  metadata: FileStartPayload,
  chunkIndex: number
): number {
  if (chunkIndex === metadata.total_chunks - 1) {
    return metadata.total_size - metadata.chunk_size * chunkIndex;
  }
  return metadata.chunk_size;
}

type MessageHandlers = {
  onEphemeralKey: (payload: EphemeralKeyPayload) => void;
  onAck: (messageId: string) => void;
  onSessionEstablished: (payload: SessionEstablishedPayload) => void;
  onMessage: (payload: MessagePayload) => void;
  onPeerOffline: (payload: PeerOfflinePayload) => void;
  onPeerDisconnected: (payload: PeerDisconnectedPayload) => void;
  onFileStart: (payload: FileStartPayload) => void;
  onFileChunk: (payload: FileChunkPayload) => void;
  onFileComplete: (payload: FileCompletePayload) => void;
  onTyping: (payload: TypingPayload) => void;
  onReaction: (payload: ReactionPayload) => void;
  onMessageDelete: (payload: MessageDeletePayload) => void;
  onMessageEdit: (payload: MessageEditPayload) => void;
  onMessageRead: (payload: MessageReadPayload) => void;
};

type MessageHandlerOptions = {
  peerId: string | null;
  handlers: MessageHandlers;
  setState: (updater: (state: ChatSessionState) => ChatSessionState) => void;
  setMessages: React.Dispatch<React.SetStateAction<ChatMessage[]>>;
  setError: (error: string | null) => void;
  setPeerActivity: (activity: 'typing' | 'voice' | 'video' | null) => void;
  clearPendingAcks: () => void;
  clearSession: () => void;
  clearFileBuffers: () => void;
  fileBuffersRef: React.MutableRefObject<Map<string, FileBuffer>>;
  typingTimeoutRef: React.MutableRefObject<number | null>;
  handleIncomingFile: (fileId: string) => void;
};

export function useMessageHandler({
  peerId,
  handlers,
  setState,
  setMessages,
  setError,
  setPeerActivity,
  clearPendingAcks,
  clearSession,
  clearFileBuffers,
  fileBuffersRef,
  typingTimeoutRef,
  handleIncomingFile,
}: MessageHandlerOptions) {
  return useCallback(
    (message: WSMessage) => {
      if (!peerId) return;

      switch (message.type) {
        case 'ephemeral_key': {
          const payload = message.payload as EphemeralKeyPayload;
          setState((currentState) => {
            if (
              currentState === 'peer_disconnected' ||
              currentState === 'peer_offline'
            ) {
              clearSession();
              return 'idle';
            }
            return currentState;
          });
          handlers.onEphemeralKey(payload);
          break;
        }

        case 'ack': {
          const payload = message.payload as AckPayload;
          handlers.onAck(payload.message_id);
          break;
        }

        case 'session_established': {
          const payload = message.payload as SessionEstablishedPayload;
          handlers.onSessionEstablished(payload);
          break;
        }

        case 'message': {
          const payload = message.payload as MessagePayload;
          handlers.onMessage(payload);
          break;
        }

        case 'peer_offline': {
          const payload = message.payload as PeerOfflinePayload;
          if (payload.peer_id === peerId) {
            clearPendingAcks();
            handlers.onPeerOffline(payload);
          }
          break;
        }

        case 'peer_disconnected': {
          const payload = message.payload as PeerDisconnectedPayload;
          if (payload.peer_id === peerId) {
            handlers.onPeerDisconnected(payload);
            setMessages([]);
            clearSession();
            clearFileBuffers();
          }
          break;
        }

        case 'file_start': {
          const payload = message.payload as FileStartPayload;
          if (payload.from === peerId) {
            if (!isValidFileMetadata(payload)) {
              setError(MESSAGES.chat.fileTransfer.errors.invalidTransfer);
              break;
            }
            if (fileBuffersRef.current.has(payload.file_id)) {
              break;
            }
            if (fileBuffersRef.current.size >= MAX_CONCURRENT_FILE_TRANSFERS) {
              setError(MESSAGES.chat.fileTransfer.errors.transferLimitExceeded);
              break;
            }
            fileBuffersRef.current.set(payload.file_id, {
              chunks: [],
              metadata: payload,
              processing: false,
              receivedSize: 0,
            });
          }
          break;
        }

        case 'file_chunk': {
          const payload = message.payload as FileChunkPayload;
          if (payload.from === peerId) {
            const buffer = fileBuffersRef.current.get(payload.file_id);
            if (buffer && !buffer.processing) {
              const maxCiphertextLength =
                Math.ceil((FILE_CHUNK_SIZE + AES_GCM_TAG_SIZE_BYTES) / 3) * 4;
              const ciphertextSize = decodedBase64Length(
                payload.ciphertext,
                maxCiphertextLength
              );
              const nonceSize = decodedBase64Length(
                payload.nonce,
                Math.ceil(AES_GCM_NONCE_SIZE_BYTES / 3) * 4
              );
              const plaintextSize =
                ciphertextSize === null
                  ? null
                  : ciphertextSize - AES_GCM_TAG_SIZE_BYTES;
              const isValidChunk =
                payload.from === buffer.metadata.from &&
                payload.to === buffer.metadata.to &&
                payload.total_chunks === buffer.metadata.total_chunks &&
                Number.isSafeInteger(payload.chunk_index) &&
                payload.chunk_index >= 0 &&
                payload.chunk_index < buffer.metadata.total_chunks &&
                buffer.chunks[payload.chunk_index] === undefined &&
                nonceSize === AES_GCM_NONCE_SIZE_BYTES &&
                plaintextSize !== null &&
                plaintextSize ===
                  expectedChunkSize(buffer.metadata, payload.chunk_index) &&
                buffer.receivedSize + plaintextSize <=
                  buffer.metadata.total_size;

              if (!isValidChunk) {
                fileBuffersRef.current.delete(payload.file_id);
                setError(MESSAGES.chat.fileTransfer.errors.invalidTransfer);
                break;
              }

              buffer.chunks[payload.chunk_index] = {
                ciphertext: payload.ciphertext,
                nonce: payload.nonce,
              };
              buffer.receivedSize += plaintextSize;
            }
          }
          break;
        }

        case 'file_complete': {
          const payload = message.payload as FileCompletePayload;
          if (payload.from === peerId) {
            const buffer = fileBuffersRef.current.get(payload.file_id);
            if (!buffer || buffer.processing) {
              break;
            }

            let hasAllChunks =
              payload.from === buffer.metadata.from &&
              payload.to === buffer.metadata.to &&
              buffer.receivedSize === buffer.metadata.total_size;
            for (let i = 0; i < buffer.metadata.total_chunks; i++) {
              if (!buffer.chunks[i]) {
                hasAllChunks = false;
                break;
              }
            }

            if (!hasAllChunks) {
              fileBuffersRef.current.delete(payload.file_id);
              setError(MESSAGES.chat.fileTransfer.errors.invalidTransfer);
              break;
            }

            void handleIncomingFile(payload.file_id);
          }
          break;
        }

        case 'typing': {
          const payload = message.payload as TypingPayload;
          if (payload.from === peerId) {
            const nextActivity = payload.is_typing
              ? (payload.activity ?? 'typing')
              : null;
            setPeerActivity(nextActivity);
            if (typingTimeoutRef.current) {
              clearTimeout(typingTimeoutRef.current);
            }
            if (payload.is_typing) {
              typingTimeoutRef.current = setTimeout(() => {
                setPeerActivity(null);
              }, 3000) as unknown as number;
            }
          }
          break;
        }

        case 'reaction': {
          const payload = message.payload as ReactionPayload;
          handlers.onReaction(payload);
          break;
        }

        case 'message_delete': {
          const payload = message.payload as MessageDeletePayload;
          handlers.onMessageDelete(payload);
          break;
        }

        case 'message_edit': {
          const payload = message.payload as MessageEditPayload;
          handlers.onMessageEdit(payload);
          break;
        }

        case 'message_read': {
          const payload = message.payload as MessageReadPayload;
          handlers.onMessageRead(payload);
          break;
        }
      }
    },
    [
      peerId,
      handlers,
      setState,
      setMessages,
      setError,
      setPeerActivity,
      clearPendingAcks,
      clearSession,
      clearFileBuffers,
      fileBuffersRef,
      typingTimeoutRef,
      handleIncomingFile,
    ]
  );
}
