import { useEffect, useRef, useState, useCallback } from 'react';
import { WebSocketClient } from '@/shared/websocket/client';
import type { WSMessage, ConnectionState } from '@/shared/websocket/types';

type UseWebSocketOptions = {
  token: string | null;
  enabled?: boolean;
  onMessage?: (message: WSMessage) => void;
  onError?: (error: Error) => void;
  onTokenExpired?: () => Promise<string | null>;
};

export function useWebSocket({
  token,
  enabled = true,
  onMessage,
  onError,
  onTokenExpired,
}: UseWebSocketOptions) {
  const [state, setState] = useState<ConnectionState>('disconnected');
  const clientRef = useRef<WebSocketClient | null>(null);
  const onMessageRef = useRef(onMessage);
  const onErrorRef = useRef(onError);

  useEffect(() => {
    onMessageRef.current = onMessage;
  }, [onMessage]);

  useEffect(() => {
    onErrorRef.current = onError;
  }, [onError]);

  useEffect(() => {
    if (!enabled || !token) {
      if (clientRef.current) {
        clientRef.current.disconnect();
        clientRef.current = null;
      }
      setState('disconnected');
      return;
    }

    const client = new WebSocketClient(token, {
      onStateChange: setState,
      onMessage: (message) => {
        onMessageRef.current?.(message);
      },
      onError: (error) => {
        onErrorRef.current?.(error);
      },
      onTokenExpired: onTokenExpired
        ? async () => {
            const newToken = await onTokenExpired();
            if (newToken && clientRef.current) {
              clientRef.current.updateToken(newToken);
            }
            return newToken;
          }
        : undefined,
    });

    clientRef.current = client;
    client.connect();

    return () => {
      client.disconnect();
      clientRef.current = null;
    };
  }, [token, enabled, onTokenExpired]);

  const send = useCallback((message: WSMessage) => {
    clientRef.current?.send(message);
  }, []);

  return {
    state,
    isConnected: state === 'connected',
    send,
  };
}
