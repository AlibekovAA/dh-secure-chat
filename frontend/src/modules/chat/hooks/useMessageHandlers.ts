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
} from '@/modules/chat/useChatSession';

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
  setPeerActivity: (activity: 'typing' | 'voice' | 'video' | null) => void;
  clearPendingAcks: () => void;
  clearSession: () => void;
  clearFileBuffers: () => void;
  fileBuffersRef: React.MutableRefObject<
    Map<
      string,
      {
        chunks: Array<{ ciphertext: string; nonce: string }>;
        metadata: FileStartPayload;
        processing: boolean;
      }
    >
  >;
  typingTimeoutRef: React.MutableRefObject<number | null>;
  handleIncomingFile: (fileId: string) => void;
};

export function useMessageHandler({
  peerId,
  handlers,
  setState,
  setMessages,
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
            fileBuffersRef.current.set(payload.file_id, {
              chunks: [],
              metadata: payload,
              processing: false,
            });
          }
          break;
        }

        case 'file_chunk': {
          const payload = message.payload as FileChunkPayload;
          if (payload.from === peerId) {
            const buffer = fileBuffersRef.current.get(payload.file_id);
            if (buffer && !buffer.processing) {
              if (
                payload.chunk_index >= 0 &&
                payload.chunk_index < payload.total_chunks
              ) {
                buffer.chunks[payload.chunk_index] = {
                  ciphertext: payload.ciphertext,
                  nonce: payload.nonce,
                };
              }
            }
          }
          break;
        }

        case 'file_complete': {
          const payload = message.payload as FileCompletePayload;
          if (payload.from === peerId) {
            setTimeout(() => {
              handleIncomingFile(payload.file_id);
            }, 100);
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
