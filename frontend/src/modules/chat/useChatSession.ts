import { useCallback, useEffect, useRef, useState } from 'react';
import { useWebSocket } from '../../shared/websocket';
import type {
  WSMessage,
  EphemeralKeyPayload,
  MessagePayload,
  SessionEstablishedPayload,
  PeerOfflinePayload,
  PeerDisconnectedPayload,
} from '../../shared/websocket/types';
import { generateEphemeralKeyPair } from '../../shared/crypto/ephemeral';
import { exportPublicKey, importPublicKey } from '../../shared/crypto/identity';
import { deriveSessionKey } from '../../shared/crypto/session';
import { encrypt, decrypt } from '../../shared/crypto/encryption';
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
  text: string;
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
            }
            break;
          }
        }
      },
      [peerId],
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
    isSessionActive: state === 'active',
  };
}
