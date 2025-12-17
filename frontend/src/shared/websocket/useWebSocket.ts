import { useEffect, useRef, useState, useCallback } from 'react';
import { WebSocketClient } from './client';
import type { WSMessage, ConnectionState } from './types';

type UseWebSocketOptions = {
  token: string | null;
  enabled?: boolean;
  onMessage?: (message: WSMessage) => void;
  onError?: (error: Error) => void;
};

export function useWebSocket({
  token,
  enabled = true,
  onMessage,
  onError,
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
    });

    clientRef.current = client;
    client.connect();

    return () => {
      client.disconnect();
      clientRef.current = null;
    };
  }, [token, enabled]);

  const send = useCallback((message: WSMessage) => {
    clientRef.current?.send(message);
  }, []);

  const disconnect = useCallback(() => {
    clientRef.current?.disconnect();
  }, []);

  const reconnect = useCallback(() => {
    if (clientRef.current) {
      clientRef.current.disconnect();
      clientRef.current.connect();
    }
  }, []);

  return {
    state,
    isConnected: state === 'connected',
    send,
    disconnect,
    reconnect,
  };
}
