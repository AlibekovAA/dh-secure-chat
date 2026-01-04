import { useCallback, useEffect, useRef, useState } from 'react';
import { useWebSocket } from '../../shared/websocket';
import type {
  WSMessage,
  EphemeralKeyPayload,
  SessionEstablishedPayload,
  PeerOfflinePayload,
  PeerDisconnectedPayload,
  FileStartPayload,
} from '../../shared/websocket/types';
import { generateEphemeralKeyPair } from '../../shared/crypto/ephemeral';
import {
  exportPublicKey,
  importPublicKey,
  loadIdentityPrivateKey,
} from '../../shared/crypto/identity';
import {
  signEphemeralKey,
  verifyEphemeralKeySignature,
} from '../../shared/crypto/signature';
import { deriveSessionKey } from '../../shared/crypto/session';
import { encrypt } from '../../shared/crypto/encryption';
import {
  encryptFile,
  calculateChunks,
  getChunkSize,
} from '../../shared/crypto/file-encryption';
import { getIdentityKey } from './api';
import type { SessionKey } from '../../shared/crypto/session';
import {
  MAX_FILE_SIZE,
  MAX_MESSAGE_LENGTH,
  MAX_VOICE_SIZE,
  EDIT_TIMEOUT_MS,
} from '../../shared/constants';
import { useAckManager } from './hooks/useAckManager';
import { useFileTransfer } from './hooks/useFileTransfer';
import { useMessageHandler } from './hooks/useMessageHandlers';
import { useIncomingMessageHandlers } from './hooks/useIncomingMessageHandlers';
import { isVoiceFile, isVideoFile, extractDurationFromFilename } from './utils';

export type ChatSessionState =
  | 'idle'
  | 'establishing'
  | 'active'
  | 'peer_offline'
  | 'peer_disconnected'
  | 'error';

export type DeliveryStatus = 'sending' | 'delivered' | 'read';

export type ChatMessage = {
  id: string;
  text?: string;
  file?: {
    filename: string;
    mimeType: string;
    size: number;
    blob?: Blob;
    accessMode?: 'download_only' | 'view_only' | 'both';
  };
  voice?: {
    filename: string;
    mimeType: string;
    size: number;
    duration: number;
    blob?: Blob;
  };
  video?: {
    filename: string;
    mimeType: string;
    size: number;
    duration: number;
    blob?: Blob;
  };
  timestamp: number;
  isOwn: boolean;
  isEdited?: boolean;
  isDeleted?: boolean;
  deliveryStatus?: DeliveryStatus;
  reactions?: Record<string, string[]>;
  replyTo?: {
    id: string;
    text?: string;
    hasFile?: boolean;
    hasVoice?: boolean;
    hasVideo?: boolean;
    isOwn?: boolean;
    isDeleted?: boolean;
  };
};

type UseChatSessionOptions = {
  token: string | null;
  peerId: string | null;
  enabled: boolean;
  onTokenExpired?: () => Promise<string | null>;
};

export function useChatSession({
  token,
  peerId,
  enabled,
  onTokenExpired,
}: UseChatSessionOptions) {
  const [state, setState] = useState<ChatSessionState>('idle');
  const [messages, setMessages] = useState<ChatMessage[]>([]);
  const [error, setError] = useState<string | null>(null);
  const [isPeerTyping, setIsPeerTyping] = useState(false);

  const sessionKeyRef = useRef<SessionKey | null>(null);
  const myEphemeralKeyRef = useRef<{
    publicKey: CryptoKey;
    privateKey: CryptoKey;
  } | null>(null);
  const peerEphemeralKeyRef = useRef<CryptoKey | null>(null);
  const messageIdCounterRef = useRef(0);
  const fileBuffersRef = useRef<
    Map<
      string,
      {
        chunks: Array<{ ciphertext: string; nonce: string }>;
        metadata: FileStartPayload;
        processing: boolean;
      }
    >
  >(new Map());
  const myIdentityPrivateKeyRef = useRef<CryptoKey | null>(null);
  const peerIdentityPublicKeyRef = useRef<string | null>(null);
  const sendRef = useRef<((message: WSMessage) => void) | null>(null);
  const pendingAcksRef = useRef<
    Map<
      string,
      {
        message: WSMessage;
        timeout: number;
        retries: number;
        timestamp: number;
      }
    >
  >(new Map());
  const typingTimeoutRef = useRef<number | null>(null);

  const pendingMessageAcksRef = useRef<Set<string>>(new Set());

  const { handleAck, sendWithAck, clearPendingAcks } = useAckManager({
    sendRef,
    pendingAcksRef,
    onMaxRetries: () => {
      setError(
        'Не удалось доставить ключ шифрования. Попробуйте переподключиться к чату.',
      );
      setState('error');
    },
  });

  const handleMessageAck = useCallback((messageId: string) => {
    setMessages((prev) => {
      const found = prev.find((msg) => msg.id === messageId && msg.isOwn);
      if (!found) {
        return prev;
      }
      return prev.map((msg) => {
        if (msg.id === messageId && msg.isOwn) {
          if (msg.deliveryStatus === 'sending' || !msg.deliveryStatus) {
            return { ...msg, deliveryStatus: 'delivered' as const };
          }
        }
        return msg;
      });
    });
    pendingMessageAcksRef.current.delete(messageId);
  }, []);

  const { handleIncomingFile, clearFileBuffers } = useFileTransfer({
    sessionKeyRef,
    fileBuffersRef,
    setMessages,
    setError,
    sendRef,
    peerId,
  });

  const {
    handleIncomingMessage,
    handleReaction,
    handleMessageDelete,
    handleMessageEdit,
    handleMessageRead,
  } = useIncomingMessageHandlers({
    sessionKeyRef,
    peerId,
    messageIdCounterRef,
    setMessages,
    setError,
    sendRef,
  });

  const clearSession = useCallback(() => {
    sessionKeyRef.current = null;
    myEphemeralKeyRef.current = null;
    peerEphemeralKeyRef.current = null;
  }, []);

  const handlePeerEphemeralKey = useCallback(
    async (payload: EphemeralKeyPayload) => {
      if (!peerId || !token) {
        return;
      }

      try {
        if (!peerIdentityPublicKeyRef.current) {
          const identityKeyResponse = await getIdentityKey(peerId);
          peerIdentityPublicKeyRef.current = identityKeyResponse.public_key;
        }

        const isValid = await verifyEphemeralKeySignature(
          payload.public_key,
          payload.signature,
          peerIdentityPublicKeyRef.current,
        );

        if (!isValid) {
          setError(
            'Ошибка проверки подписи ключа. Возможна попытка атаки. Переподключитесь.',
          );
          setState('error');
          return;
        }

        if (payload.requires_ack && payload.from) {
          sendRef.current?.({
            type: 'ack',
            payload: {
              to: payload.from,
              message_id: payload.message_id,
            },
          });
        }

        const peerPublicKey = await importPublicKey(payload.public_key);
        peerEphemeralKeyRef.current = peerPublicKey;

        if (myEphemeralKeyRef.current) {
          const sessionKey = await deriveSessionKey(
            myEphemeralKeyRef.current.privateKey,
            peerPublicKey,
          );

          sessionKeyRef.current = sessionKey;

          sendRef.current?.({
            type: 'session_established',
            payload: {
              to: peerId,
              peer_id: peerId,
            },
          });

          setState((currentState) => {
            if (currentState !== 'active') {
              return 'active';
            }
            return currentState;
          });
          setError(null);
        }
      } catch (err) {
        setError(
          'Не удалось установить защищенную сессию. Попробуйте переподключиться.',
        );
        setState('error');
      }
    },
    [peerId, token],
  );

  const handlePeerOffline = useCallback((_payload: PeerOfflinePayload) => {
    setState('peer_offline');
    setError('Собеседник не в сети. Сообщение не может быть доставлено.');
  }, []);

  const handlePeerDisconnected = useCallback(
    (_payload: PeerDisconnectedPayload) => {
      setState('peer_disconnected');
    },
    [],
  );

  const handleSessionEstablished = useCallback(
    (payload: SessionEstablishedPayload) => {
      if (payload.peer_id === peerId && sessionKeyRef.current) {
        setState('active');
        setError(null);
      }
    },
    [peerId],
  );

  const messageHandler = useMessageHandler({
    peerId,
    handlers: {
      onEphemeralKey: handlePeerEphemeralKey,
      onAck: (messageId: string) => {
        handleAck(messageId);
        if (!messageId.startsWith('ephemeral-')) {
          handleMessageAck(messageId);
        }
      },
      onSessionEstablished: handleSessionEstablished,
      onMessage: handleIncomingMessage,
      onPeerOffline: handlePeerOffline,
      onPeerDisconnected: handlePeerDisconnected,
      onFileStart: () => {},
      onFileChunk: () => {},
      onFileComplete: () => {},
      onTyping: () => {},
      onReaction: handleReaction,
      onMessageDelete: handleMessageDelete,
      onMessageEdit: handleMessageEdit,
      onMessageRead: handleMessageRead,
    },
    setState,
    setMessages,
    setIsPeerTyping,
    clearPendingAcks,
    clearSession,
    clearFileBuffers,
    fileBuffersRef,
    typingTimeoutRef,
    handleIncomingFile,
  });

  const { isConnected, send } = useWebSocket({
    token,
    enabled: enabled && !!token,
    onMessage: messageHandler,
    onError: useCallback((err: Error) => {
      setError(err.message);
      setState('error');
    }, []),
    onTokenExpired,
  });

  useEffect(() => {
    sendRef.current = send;
  }, [send]);

  const startSession = useCallback(async () => {
    if (!token || !peerId || !isConnected) return;

    setState('establishing');
    setError(null);
    setMessages([]);
    sessionKeyRef.current = null;
    myEphemeralKeyRef.current = null;
    peerEphemeralKeyRef.current = null;
    peerIdentityPublicKeyRef.current = null;

    clearPendingAcks();

    try {
      if (!myIdentityPrivateKeyRef.current) {
        const identityKey = await loadIdentityPrivateKey();
        if (!identityKey) {
          setError(
            'Не найден приватный ключ. Пожалуйста, перезайдите в систему.',
          );
          setState('error');
          return;
        }
        myIdentityPrivateKeyRef.current = identityKey;
      }

      const myEphemeral = await generateEphemeralKeyPair();
      myEphemeralKeyRef.current = myEphemeral;

      const myEphemeralPublicKeyBase64 = await exportPublicKey(
        myEphemeral.publicKey,
      );

      const signature = await signEphemeralKey(
        myEphemeralPublicKeyBase64,
        myIdentityPrivateKeyRef.current,
      );

      const messageId = `ephemeral-${Date.now()}-${Math.random()
        .toString(36)
        .slice(2, 11)}`;

      sendWithAck(
        {
          type: 'ephemeral_key',
          payload: {
            to: peerId,
            public_key: myEphemeralPublicKeyBase64,
            signature,
            message_id: messageId,
            requires_ack: true,
          },
        },
        true,
      );

      if (peerEphemeralKeyRef.current) {
        const sessionKey = await deriveSessionKey(
          myEphemeral.privateKey,
          peerEphemeralKeyRef.current,
        );

        sessionKeyRef.current = sessionKey;

        send({
          type: 'session_established',
          payload: {
            to: peerId,
            peer_id: peerId,
          },
        });

        setState('active');
        setError(null);
      }
    } catch (err) {
      const errorMessage =
        err instanceof Error ? err.message : 'Ошибка установки сессии';
      setError(
        `Ошибка установки защищенной сессии: ${errorMessage}. Попробуйте переподключиться.`,
      );
      setState('error');
    }
  }, [token, peerId, isConnected, send, sendWithAck]);

  const sendMessage = useCallback(
    async (text: string, replyToMessageId?: string) => {
      if (
        !sessionKeyRef.current ||
        !peerId ||
        !isConnected ||
        state !== 'active'
      ) {
        return;
      }

      const trimmed = text.trim();
      if (!trimmed) {
        return;
      }

      if (trimmed.length > MAX_MESSAGE_LENGTH) {
        setError(
          `Сообщение слишком длинное (максимум ${MAX_MESSAGE_LENGTH} символов)`,
        );
        return;
      }

      try {
        const { ciphertext, nonce } = await encrypt(
          sessionKeyRef.current,
          trimmed,
        );

        const messageId = `msg-${Date.now()}-${messageIdCounterRef.current++}`;

        send({
          type: 'message',
          payload: {
            to: peerId,
            message_id: messageId,
            ciphertext,
            nonce,
            reply_to_message_id: replyToMessageId,
          },
        });

        setMessages((prev) => {
          let replyTo: ChatMessage['replyTo'] | undefined;
          if (replyToMessageId) {
            const target = prev.find((m) => m.id === replyToMessageId);
            if (target) {
              replyTo = {
                id: target.id,
                text: target.text,
                hasFile: !!target.file,
                hasVoice: !!target.voice,
                isOwn: target.isOwn,
                isDeleted: target.isDeleted,
              };
            }
          }

          const newMessage: ChatMessage = {
            id: messageId,
            text: trimmed,
            timestamp: Date.now(),
            isOwn: true,
            deliveryStatus: 'sending',
            replyTo,
          };

          pendingMessageAcksRef.current.add(messageId);

          return [...prev, newMessage];
        });
      } catch (err) {
        setError(
          'Не удалось отправить сообщение. Проверьте соединение и попробуйте снова.',
        );
      }
    },
    [peerId, isConnected, state, send],
  );

  const sendFile = useCallback(
    async (
      file: File,
      accessMode: 'download_only' | 'view_only' | 'both' = 'both',
      voiceDuration?: number,
      videoDuration?: number,
    ) => {
      if (
        !sessionKeyRef.current ||
        !peerId ||
        !isConnected ||
        state !== 'active'
      ) {
        setError(
          'Не удалось отправить файл: защищенная сессия не активна. Дождитесь установки соединения.',
        );
        return;
      }

      if (file.size > MAX_FILE_SIZE) {
        setError(
          'Файл слишком большой. Максимальный размер: 50MB. Выберите файл меньшего размера.',
        );
        return;
      }

      if (file.size === 0) {
        throw new Error('Файл пустой. Выберите файл с содержимым.');
      }

      try {
        const fileId = `file-${Date.now()}-${Math.random()
          .toString(36)
          .slice(2, 11)}`;
        const totalChunks = calculateChunks(file.size);
        const chunkSize = getChunkSize();

        let chunks: Array<{ ciphertext: string; nonce: string }>;
        let totalSize: number;
        try {
          const result = await encryptFile(sessionKeyRef.current, file);
          chunks = result.chunks;
          totalSize = result.totalSize;
        } catch (encryptError) {
          setError(
            `Ошибка шифрования файла: ${
              encryptError instanceof Error
                ? encryptError.message
                : 'неизвестная ошибка'
            }. Попробуйте выбрать другой файл.`,
          );
          return;
        }

        const isVoice = isVoiceFile(file.type || '');
        const isVideo = !isVoice && isVideoFile(file.type || '');
        let mimeType =
          file.type ||
          (isVoice
            ? 'audio/webm'
            : isVideo
            ? 'video/webm'
            : 'application/octet-stream');

        if ((isVoice || isVideo) && mimeType.includes(';')) {
          mimeType = mimeType.split(';')[0].trim();
        }

        send({
          type: 'file_start',
          payload: {
            to: peerId,
            file_id: fileId,
            filename: file.name,
            mime_type: mimeType,
            total_size: totalSize,
            total_chunks: totalChunks,
            chunk_size: chunkSize,
            access_mode: accessMode,
          },
        });

        for (let i = 0; i < chunks.length; i++) {
          const chunkMsg = {
            type: 'file_chunk' as const,
            payload: {
              to: peerId,
              file_id: fileId,
              chunk_index: i,
              total_chunks: totalChunks,
              ciphertext: chunks[i].ciphertext,
              nonce: chunks[i].nonce,
            },
          };

          if (i > 0 && i % 5 === 0) {
            await new Promise((resolve) => setTimeout(resolve, 10));
          }

          send(chunkMsg);
        }

        send({
          type: 'file_complete',
          payload: {
            to: peerId,
            file_id: fileId,
          },
        });

        const blobClone = new Blob([file], { type: mimeType });
        const extractedDuration = extractDurationFromFilename(file.name);
        const finalDuration =
          voiceDuration !== undefined
            ? voiceDuration
            : videoDuration !== undefined
            ? videoDuration
            : extractedDuration;

        const newMessage: ChatMessage = {
          id: fileId,
          ...(isVoice
            ? {
                voice: {
                  filename: file.name,
                  mimeType,
                  size: file.size,
                  duration: finalDuration,
                  blob: blobClone,
                },
              }
            : isVideo
            ? {
                video: {
                  filename: file.name,
                  mimeType,
                  size: file.size,
                  duration: finalDuration,
                  blob: blobClone,
                },
              }
            : {
                file: {
                  filename: file.name,
                  mimeType,
                  size: file.size,
                  blob: blobClone,
                  accessMode,
                },
              }),
          timestamp: Date.now(),
          isOwn: true,
          deliveryStatus: 'sending',
        };

        pendingMessageAcksRef.current.add(fileId);

        setMessages((prev) => [...prev, newMessage]);
      } catch (err) {
        const errorMsg =
          err instanceof Error ? err.message : 'Ошибка отправки файла';
        setError(
          `Не удалось отправить файл: ${errorMsg}. Проверьте соединение и попробуйте снова.`,
        );
        throw err;
      }
    },
    [peerId, isConnected, state, send],
  );

  useEffect(() => {
    if (enabled && peerId && isConnected && state === 'idle') {
      startSession();
    } else if (
      enabled &&
      peerId &&
      isConnected &&
      state === 'peer_disconnected'
    ) {
      setState('idle');
      startSession();
    } else if (!enabled || !peerId) {
      setState('idle');
      setMessages([]);
      setError(null);
      sessionKeyRef.current = null;
      myEphemeralKeyRef.current = null;
      peerEphemeralKeyRef.current = null;
      peerIdentityPublicKeyRef.current = null;

      clearPendingAcks();
    }
  }, [enabled, peerId, isConnected, state, startSession]);

  useEffect(() => {
    return () => {
      clearPendingAcks();
      clearFileBuffers();
      clearSession();
      peerIdentityPublicKeyRef.current = null;
    };
  }, [clearPendingAcks, clearFileBuffers, clearSession]);

  const sendVoice = useCallback(
    async (file: File, duration: number) => {
      if (file.size === 0) {
        throw new Error(
          'Голосовое сообщение пустое. Запишите сообщение с звуком.',
        );
      }
      if (file.size > MAX_VOICE_SIZE) {
        throw new Error(
          'Голосовое сообщение слишком большое. Максимальный размер: 10MB. Запишите более короткое сообщение.',
        );
      }
      await sendFile(file, 'both', duration);
    },
    [sendFile],
  );

  const sendTyping = useCallback(
    (isTyping: boolean) => {
      if (!peerId || !isConnected || !send) return;
      send({
        type: 'typing',
        payload: {
          to: peerId,
          is_typing: isTyping,
        },
      });
    },
    [peerId, isConnected, send],
  );

  useEffect(() => {
    return () => {
      if (typingTimeoutRef.current) {
        clearTimeout(typingTimeoutRef.current);
      }
    };
  }, []);

  const sendReaction = useCallback(
    async (messageId: string, emoji: string, action: 'add' | 'remove') => {
      if (!peerId || !isConnected || !send || state !== 'active') return;

      const message = messages.find((m) => m.id === messageId);
      if (!message) return;

      const reactions = message.reactions || {};
      const emojiReactions = reactions[emoji] || [];
      const myUserId = localStorage.getItem('userId') || '';

      if (action === 'add' && emojiReactions.includes(myUserId)) return;
      if (action === 'remove' && !emojiReactions.includes(myUserId)) return;

      send({
        type: 'reaction',
        payload: {
          to: peerId,
          message_id: messageId,
          emoji,
          action,
        },
      });

      setMessages((prev) =>
        prev.map((msg) => {
          if (msg.id === messageId) {
            const currentReactions = msg.reactions || {};
            const currentEmojiReactions = currentReactions[emoji] || [];
            if (action === 'add') {
              return {
                ...msg,
                reactions: {
                  ...currentReactions,
                  [emoji]: [...currentEmojiReactions, myUserId],
                },
              };
            } else {
              return {
                ...msg,
                reactions: {
                  ...currentReactions,
                  [emoji]: currentEmojiReactions.filter(
                    (id) => id !== myUserId,
                  ),
                },
              };
            }
          }
          return msg;
        }),
      );
    },
    [peerId, isConnected, send, state, messages],
  );

  const deleteMessage = useCallback(
    (messageId: string, scope: 'me' | 'all') => {
      if (!peerId || !isConnected || !send || state !== 'active') return;

      const message = messages.find((m) => m.id === messageId);
      if (!message || !message.isOwn) return;

      if (scope === 'all') {
        send({
          type: 'message_delete',
          payload: {
            to: peerId,
            message_id: messageId,
            scope: 'all',
          },
        });
      }

      setMessages((prev) =>
        prev.map((msg) => {
          if (msg.id === messageId) {
            return { ...msg, isDeleted: true, text: undefined };
          }

          if (msg.replyTo?.id === messageId) {
            return {
              ...msg,
              replyTo: {
                ...msg.replyTo,
                isDeleted: true,
                text: undefined,
                hasFile: false,
                hasVoice: false,
              },
            };
          }

          return msg;
        }),
      );
    },
    [peerId, isConnected, send, state, messages],
  );

  const editMessage = useCallback(
    async (messageId: string, newText: string) => {
      if (
        !sessionKeyRef.current ||
        !peerId ||
        !isConnected ||
        !send ||
        state !== 'active'
      ) {
        return;
      }

      const message = messages.find((m) => m.id === messageId);
      if (!message || !message.isOwn || !message.text) return;

      const timeSinceSent = Date.now() - message.timestamp;
      const EDIT_TIMEOUT = EDIT_TIMEOUT_MS;
      if (timeSinceSent > EDIT_TIMEOUT) {
        setError(
          'Редактирование доступно только в течение 15 минут после отправки.',
        );
        return;
      }

      const trimmed = newText.trim();
      if (!trimmed || trimmed === message.text) return;

      if (trimmed.length > MAX_MESSAGE_LENGTH) {
        setError(
          `Сообщение слишком длинное (максимум ${MAX_MESSAGE_LENGTH} символов)`,
        );
        return;
      }

      try {
        const { ciphertext, nonce } = await encrypt(
          sessionKeyRef.current,
          trimmed,
        );

        send({
          type: 'message_edit',
          payload: {
            to: peerId,
            message_id: messageId,
            ciphertext,
            nonce,
          },
        });

        setMessages((prev) =>
          prev.map((msg) => {
            if (msg.id === messageId) {
              return {
                ...msg,
                text: trimmed,
                isEdited: true,
              };
            }

            if (msg.replyTo?.id === messageId) {
              return {
                ...msg,
                replyTo: {
                  ...msg.replyTo,
                  text: trimmed,
                },
              };
            }

            return msg;
          }),
        );
      } catch (err) {
        setError('Не удалось отредактировать сообщение. Попробуйте снова.');
      }
    },
    [peerId, isConnected, send, state, messages, sessionKeyRef],
  );

  const markMessageAsRead = useCallback(
    (messageId: string) => {
      if (!peerId || !isConnected || !send || state !== 'active') return;

      const message = messages.find((m) => m.id === messageId);
      if (!message || message.isOwn) return;

      send({
        type: 'message_read',
        payload: {
          to: peerId,
          message_id: messageId,
        },
      });
    },
    [peerId, isConnected, send, state, messages],
  );

  return {
    state,
    messages,
    error,
    sendMessage,
    sendFile,
    sendVoice,
    sendTyping,
    sendReaction,
    deleteMessage,
    editMessage,
    markMessageAsRead,
    isPeerTyping,
    isSessionActive: state === 'active',
  };
}
