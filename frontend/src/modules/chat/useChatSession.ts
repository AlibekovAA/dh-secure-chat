import { useCallback, useEffect, useRef, useState } from 'react';
import { useWebSocket } from '../../shared/websocket';
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
import { encrypt, decrypt } from '../../shared/crypto/encryption';
import {
  encryptFile,
  decryptFile,
  calculateChunks,
  getChunkSize,
} from '../../shared/crypto/file-encryption';
import { getIdentityKey } from './api';
import type { SessionKey } from '../../shared/crypto/session';
import { MAX_FILE_SIZE } from './constants';

export type ChatSessionState =
  | 'idle'
  | 'establishing'
  | 'active'
  | 'peer_offline'
  | 'peer_disconnected'
  | 'error';

export type ChatMessage = {
  id: string;
  text?: string;
  file?: {
    filename: string;
    mimeType: string;
    size: number;
    blob?: Blob;
  };
  timestamp: number;
  isOwn: boolean;
};

type UseChatSessionOptions = {
  token: string | null;
  peerId: string | null;
  enabled: boolean;
};

export function useChatSession({
  token,
  peerId,
  enabled,
}: UseChatSessionOptions) {
  const [state, setState] = useState<ChatSessionState>('idle');
  const [messages, setMessages] = useState<ChatMessage[]>([]);
  const [error, setError] = useState<string | null>(null);

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
  const ACK_TIMEOUT = 5000;
  const MAX_RETRIES = 3;
  const RETRY_DELAY = 1000;

  const handleAck = useCallback((messageId: string) => {
    const pending = pendingAcksRef.current.get(messageId);
    if (pending) {
      clearTimeout(pending.timeout);
      pendingAcksRef.current.delete(messageId);
    }
  }, []);

  const scheduleRetry = useCallback((messageId: string) => {
    const pending = pendingAcksRef.current.get(messageId);
    if (!pending) return;

    pending.retries++;
    if (pending.retries >= MAX_RETRIES) {
      pendingAcksRef.current.delete(messageId);
      setError('Не удалось доставить ключ. Попробуйте переподключиться.');
      setState('error');
      return;
    }

    const retryTimeout = setTimeout(() => {
      sendRef.current?.(pending.message);
      const ackTimeout = setTimeout(() => {
        scheduleRetry(messageId);
      }, ACK_TIMEOUT) as unknown as number;
      pending.timeout = ackTimeout;
    }, RETRY_DELAY) as unknown as number;

    pending.timeout = retryTimeout;
  }, []);

  const sendWithAck = useCallback(
    (message: WSMessage, requiresAck: boolean) => {
      if (!requiresAck) {
        sendRef.current?.(message);
        return;
      }

      const payload = message.payload as EphemeralKeyPayload;
      const messageId = payload.message_id;

      const timeout = setTimeout(() => {
        scheduleRetry(messageId);
      }, ACK_TIMEOUT) as unknown as number;

      pendingAcksRef.current.set(messageId, {
        message,
        timeout,
        retries: 0,
        timestamp: Date.now(),
      });

      sendRef.current?.(message);
    },
    [scheduleRetry],
  );

  const handlePeerEphemeralKey = useCallback(
    async (payload: EphemeralKeyPayload) => {
      if (!peerId || !token) {
        return;
      }

      try {
        if (!peerIdentityPublicKeyRef.current) {
          const identityKeyResponse = await getIdentityKey(peerId, token);
          peerIdentityPublicKeyRef.current = identityKeyResponse.public_key;
        }

        const isValid = await verifyEphemeralKeySignature(
          payload.public_key,
          payload.signature,
          peerIdentityPublicKeyRef.current,
        );

        if (!isValid) {
          setError('Ошибка проверки подписи ключа');
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
        setError('Ошибка установки сессии');
        setState('error');
      }
    },
    [peerId, token],
  );

  const handleIncomingMessage = useCallback(async (payload: MessagePayload) => {
    if (!sessionKeyRef.current) return;

    try {
      const decrypted = await decrypt(
        sessionKeyRef.current,
        payload.ciphertext,
        payload.nonce,
      );

      const newMessage: ChatMessage = {
        id: `msg-${Date.now()}-${messageIdCounterRef.current++}`,
        text: decrypted,
        timestamp: Date.now(),
        isOwn: false,
      };

      setMessages((prev) => [...prev, newMessage]);
    } catch (err) {
      setError('Ошибка расшифровки сообщения');
    }
  }, []);

  const handleIncomingFile = useCallback(async (fileId: string) => {
    if (!sessionKeyRef.current) {
      return;
    }

    const buffer = fileBuffersRef.current.get(fileId);
    if (!buffer) {
      return;
    }

    const { chunks, metadata } = buffer;
    const expectedChunks = metadata.total_chunks;

    const sortedChunks: Array<{ ciphertext: string; nonce: string }> = [];
    for (let i = 0; i < expectedChunks; i++) {
      const chunk = chunks[i];
      if (!chunk || !chunk.ciphertext || !chunk.nonce) {
        setError(
          `Не все части файла получены (${sortedChunks.length}/${expectedChunks})`,
        );
        return;
      }
      sortedChunks.push(chunk);
    }

    try {
      const blob = await decryptFile(sessionKeyRef.current, sortedChunks);

      const newMessage: ChatMessage = {
        id: `file-${Date.now()}-${messageIdCounterRef.current++}`,
        file: {
          filename: metadata.filename,
          mimeType: metadata.mime_type,
          size: metadata.total_size,
          blob,
        },
        timestamp: Date.now(),
        isOwn: false,
      };

      setMessages((prev) => [...prev, newMessage]);
      fileBuffersRef.current.delete(fileId);
    } catch (err) {
      setError('Ошибка расшифровки файла');
      fileBuffersRef.current.delete(fileId);
    }
  }, []);

  const { isConnected, send } = useWebSocket({
    token,
    enabled: enabled && !!token,
    onMessage: useCallback(
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
                sessionKeyRef.current = null;
                myEphemeralKeyRef.current = null;
                peerEphemeralKeyRef.current = null;
                return 'idle';
              }
              return currentState;
            });
            handlePeerEphemeralKey(payload);
            break;
          }

          case 'ack': {
            const payload = message.payload as AckPayload;
            handleAck(payload.message_id);
            break;
          }

          case 'session_established': {
            const payload = message.payload as SessionEstablishedPayload;
            if (payload.peer_id === peerId && sessionKeyRef.current) {
              setState('active');
              setError(null);
            }
            break;
          }

          case 'message': {
            const payload = message.payload as MessagePayload;
            handleIncomingMessage(payload);
            break;
          }

          case 'peer_offline': {
            const payload = message.payload as PeerOfflinePayload;
            if (payload.peer_id === peerId) {
              for (const [, pending] of pendingAcksRef.current) {
                clearTimeout(pending.timeout);
              }
              pendingAcksRef.current.clear();

              setState('peer_offline');
              setError('Собеседник не в сети');
            }
            break;
          }

          case 'peer_disconnected': {
            const payload = message.payload as PeerDisconnectedPayload;
            if (payload.peer_id === peerId) {
              setState('peer_disconnected');
              setMessages([]);
              sessionKeyRef.current = null;
              myEphemeralKeyRef.current = null;
              peerEphemeralKeyRef.current = null;
              fileBuffersRef.current.clear();
            }
            break;
          }

          case 'file_start': {
            const payload = message.payload as FileStartPayload;
            if (payload.from === peerId) {
              fileBuffersRef.current.set(payload.file_id, {
                chunks: [],
                metadata: payload,
              });
            }
            break;
          }

          case 'file_chunk': {
            const payload = message.payload as FileChunkPayload;
            if (payload.from === peerId) {
              const buffer = fileBuffersRef.current.get(payload.file_id);
              if (buffer) {
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
              handleIncomingFile(payload.file_id);
            }
            break;
          }
        }
      },
      [
        peerId,
        handleIncomingFile,
        handleIncomingMessage,
        handlePeerEphemeralKey,
        handleAck,
      ],
    ),
    onError: useCallback((err: Error) => {
      setError(err.message);
      setState('error');
    }, []),
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

    for (const [, pending] of pendingAcksRef.current) {
      clearTimeout(pending.timeout);
    }
    pendingAcksRef.current.clear();

    try {
      if (!myIdentityPrivateKeyRef.current) {
        const identityKey = await loadIdentityPrivateKey();
        if (!identityKey) {
          setError('Не найден приватный ключ');
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
      setError(`Ошибка установки сессии: ${errorMessage}`);
      setState('error');
    }
  }, [token, peerId, isConnected, send, sendWithAck]);

  const sendMessage = useCallback(
    async (text: string) => {
      if (
        !sessionKeyRef.current ||
        !peerId ||
        !isConnected ||
        state !== 'active'
      ) {
        return;
      }

      try {
        const { ciphertext, nonce } = await encrypt(
          sessionKeyRef.current,
          text,
        );

        send({
          type: 'message',
          payload: {
            to: peerId,
            ciphertext,
            nonce,
          },
        });

        const newMessage: ChatMessage = {
          id: `msg-${Date.now()}-${messageIdCounterRef.current++}`,
          text,
          timestamp: Date.now(),
          isOwn: true,
        };

        setMessages((prev) => [...prev, newMessage]);
      } catch (err) {
        setError('Ошибка отправки сообщения');
      }
    },
    [peerId, isConnected, state, send],
  );

  const sendFile = useCallback(
    async (file: File) => {
      if (
        !sessionKeyRef.current ||
        !peerId ||
        !isConnected ||
        state !== 'active'
      ) {
        setError('Не удалось отправить файл: сессия не активна');
        return;
      }

      if (file.size > MAX_FILE_SIZE) {
        setError('Файл слишком большой (максимум 50MB)');
        return;
      }

      if (file.size === 0) {
        setError('Файл пустой');
        return;
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
                : 'Неизвестная ошибка'
            }`,
          );
          return;
        }

        send({
          type: 'file_start',
          payload: {
            to: peerId,
            file_id: fileId,
            filename: file.name,
            mime_type: file.type || 'application/octet-stream',
            total_size: totalSize,
            total_chunks: totalChunks,
            chunk_size: chunkSize,
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

        const newMessage: ChatMessage = {
          id: `file-${Date.now()}-${messageIdCounterRef.current++}`,
          file: {
            filename: file.name,
            mimeType: file.type || 'application/octet-stream',
            size: file.size,
          },
          timestamp: Date.now(),
          isOwn: true,
        };

        setMessages((prev) => [...prev, newMessage]);
      } catch (err) {
        setError('Ошибка отправки файла');
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

      for (const [, pending] of pendingAcksRef.current) {
        clearTimeout(pending.timeout);
      }
      pendingAcksRef.current.clear();
    }
  }, [enabled, peerId, isConnected, state, startSession]);

  return {
    state,
    messages,
    error,
    sendMessage,
    sendFile,
    isSessionActive: state === 'active',
  };
}
