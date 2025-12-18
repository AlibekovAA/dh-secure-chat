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
} from '../../shared/websocket/types';
import { generateEphemeralKeyPair } from '../../shared/crypto/ephemeral';
import { exportPublicKey, importPublicKey } from '../../shared/crypto/identity';
import { deriveSessionKey } from '../../shared/crypto/session';
import { encrypt, decrypt } from '../../shared/crypto/encryption';
import {
  encryptFile,
  decryptFile,
  calculateChunks,
  getChunkSize,
} from '../../shared/crypto/file-encryption';
import type { SessionKey } from '../../shared/crypto/session';

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
  peerUsername: string;
  enabled: boolean;
};

export function useChatSession({
  token,
  peerId,
  peerUsername,
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
      console.warn('handleIncomingFile: no session key', fileId);
      return;
    }

    const buffer = fileBuffersRef.current.get(fileId);
    if (!buffer) {
      console.warn('handleIncomingFile: no buffer for file', fileId);
      return;
    }

    const { chunks, metadata } = buffer;
    const expectedChunks = metadata.total_chunks;

    const sortedChunks: Array<{ ciphertext: string; nonce: string }> = [];
    for (let i = 0; i < expectedChunks; i++) {
      const chunk = chunks[i];
      if (!chunk || !chunk.ciphertext || !chunk.nonce) {
        console.warn(
          `handleIncomingFile: missing chunk ${i} of ${expectedChunks} for file`,
          fileId,
        );
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
      console.error('handleIncomingFile: decrypt error', err);
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
            handlePeerEphemeralKey(payload.public_key);
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
            if (payload.to === peerId) {
              fileBuffersRef.current.set(payload.file_id, {
                chunks: [],
                metadata: payload,
              });
              console.log(
                'file_start received',
                payload.file_id,
                payload.total_chunks,
              );
            }
            break;
          }

          case 'file_chunk': {
            const payload = message.payload as FileChunkPayload;
            if (payload.to === peerId) {
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
                  const received = buffer.chunks.filter(
                    (c) => c !== undefined,
                  ).length;
                  console.log(
                    `file_chunk received ${payload.chunk_index + 1}/${
                      payload.total_chunks
                    } (${received} total)`,
                    payload.file_id,
                  );
                }
              } else {
                console.warn('file_chunk: no buffer for', payload.file_id);
              }
            }
            break;
          }

          case 'file_complete': {
            const payload = message.payload as FileCompletePayload;
            if (payload.to === peerId) {
              console.log('file_complete received', payload.file_id);
              handleIncomingFile(payload.file_id);
            }
            break;
          }
        }
      },
      [peerId, handleIncomingFile, handleIncomingMessage],
    ),
    onError: useCallback((err: Error) => {
      setError(err.message);
      setState('error');
    }, []),
  });

  const handlePeerEphemeralKey = useCallback(
    async (peerPublicKeyBase64: string) => {
      if (!peerId) {
        return;
      }

      try {
        const peerPublicKey = await importPublicKey(peerPublicKeyBase64);
        peerEphemeralKeyRef.current = peerPublicKey;

        if (myEphemeralKeyRef.current) {
          const sessionKey = await deriveSessionKey(
            myEphemeralKeyRef.current.privateKey,
            peerPublicKey,
          );

          sessionKeyRef.current = sessionKey;

          send({
            type: 'session_established',
            payload: {
              to: peerId,
              peer_id: peerId,
            },
          });

          if (state !== 'active') {
            setState('active');
            setError(null);
          }
        }
      } catch (err) {
        setError('Ошибка установки сессии');
        setState('error');
      }
    },
    [peerId, send, state],
  );

  const startSession = useCallback(async () => {
    if (!token || !peerId || !isConnected) return;

    setState('establishing');
    setError(null);
    setMessages([]);
    sessionKeyRef.current = null;
    myEphemeralKeyRef.current = null;
    peerEphemeralKeyRef.current = null;

    try {
      const myEphemeral = await generateEphemeralKeyPair();
      myEphemeralKeyRef.current = myEphemeral;

      const myEphemeralPublicKeyBase64 = await exportPublicKey(
        myEphemeral.publicKey,
      );

      send({
        type: 'ephemeral_key',
        payload: {
          to: peerId,
          public_key: myEphemeralPublicKeyBase64,
        },
      });

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
      setError('Ошибка установки сессии');
      setState('error');
    }
  }, [token, peerId, isConnected, send]);

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
        return;
      }

      const MAX_FILE_SIZE = 50 * 1024 * 1024;
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
          .substr(2, 9)}`;
        const totalChunks = calculateChunks(file.size);
        const chunkSize = getChunkSize();

        const { chunks, totalSize } = await encryptFile(
          sessionKeyRef.current,
          file,
        );

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
          send({
            type: 'file_chunk',
            payload: {
              to: peerId,
              file_id: fileId,
              chunk_index: i,
              total_chunks: totalChunks,
              ciphertext: chunks[i].ciphertext,
              nonce: chunks[i].nonce,
            },
          });
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
