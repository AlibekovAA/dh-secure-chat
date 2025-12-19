import type { WSMessage, ConnectionState } from './types';

type MessageHandler = (message: WSMessage) => void;

type EventHandlers = {
  onStateChange?: (state: ConnectionState) => void;
  onMessage?: MessageHandler;
  onError?: (error: Error) => void;
};

export class WebSocketClient {
  private ws: WebSocket | null = null;
  private token: string;
  private url: string;
  private handlers: EventHandlers;
  private reconnectAttempts = 0;
  private maxReconnectAttempts = 5;
  private baseDelay = 1000;
  private reconnectTimer: number | null = null;
  private state: ConnectionState = 'disconnected';

  constructor(token: string, handlers: EventHandlers = {}) {
    this.token = token;
    const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
    const host = window.location.host;
    this.url = `${protocol}//${host}/ws/`;
    this.handlers = handlers;
  }

  connect(): void {
    if (this.ws?.readyState === WebSocket.OPEN) {
      return;
    }

    if (this.ws?.readyState === WebSocket.CONNECTING) {
      return;
    }

    this.setState('connecting');

    try {
      this.ws = new WebSocket(this.url);

      this.ws.onopen = () => {
        this.ws?.send(
          JSON.stringify({
            type: 'auth',
            payload: { token: this.token },
          }),
        );
      };

      this.ws.onmessage = (event) => {
        try {
          const data = event.data as string;
          const messages = data
            .split('\n')
            .filter((line) => line.trim() !== '');

          for (const messageData of messages) {
            try {
              const message = JSON.parse(messageData) as WSMessage;

              if (message.type === 'auth') {
                const payload = message.payload as {
                  authenticated?: boolean;
                  error?: string;
                };
                if (payload.authenticated === true) {
                  this.setState('connected');
                  this.reconnectAttempts = 0;
                  continue;
                }
                if (payload.error) {
                  this.handlers.onError?.(new Error(payload.error));
                  this.ws?.close();
                  return;
                }
              }

              if (this.state === 'connecting') {
                this.setState('connected');
                this.reconnectAttempts = 0;
              }

              this.handlers.onMessage?.(message);
            } catch (parseErr) {
              console.error(
                'Failed to parse WebSocket message:',
                parseErr,
                'Data:',
                messageData,
              );
            }
          }
        } catch (err) {
          this.handlers.onError?.(
            new Error('Failed to parse WebSocket message'),
          );
        }
      };

      this.ws.onerror = () => {
        this.setState('error');
        this.handlers.onError?.(new Error('WebSocket connection error'));
      };

      this.ws.onclose = () => {
        this.setState('disconnected');
        this.ws = null;

        if (this.reconnectAttempts < this.maxReconnectAttempts) {
          this.scheduleReconnect();
        } else {
          this.handlers.onError?.(
            new Error('Max reconnection attempts reached'),
          );
        }
      };
    } catch (err) {
      this.setState('error');
      this.handlers.onError?.(
        err instanceof Error ? err : new Error('Failed to create WebSocket'),
      );
    }
  }

  disconnect(): void {
    if (this.reconnectTimer !== null) {
      clearTimeout(this.reconnectTimer);
      this.reconnectTimer = null;
    }

    if (this.ws) {
      this.ws.close();
      this.ws = null;
    }

    this.setState('disconnected');
    this.reconnectAttempts = this.maxReconnectAttempts;
  }

  send(message: WSMessage): void {
    if (this.ws?.readyState === WebSocket.OPEN) {
      this.ws.send(JSON.stringify(message));
    } else {
      this.handlers.onError?.(new Error('WebSocket is not connected'));
    }
  }

  private setState(state: ConnectionState): void {
    if (this.state !== state) {
      this.state = state;
      this.handlers.onStateChange?.(state);
    }
  }

  private scheduleReconnect(): void {
    if (this.reconnectTimer !== null) {
      return;
    }

    this.reconnectAttempts++;
    const delay = Math.min(
      this.baseDelay * Math.pow(2, this.reconnectAttempts - 1),
      30000,
    );

    this.reconnectTimer = setTimeout(() => {
      this.reconnectTimer = null;
      this.connect();
    }, delay) as unknown as number;
  }
}
