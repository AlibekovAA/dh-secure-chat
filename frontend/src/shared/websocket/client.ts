import type { WSMessage, ConnectionState } from './types';
import { SequenceManager } from './sequence-manager';

type MessageHandler = (message: WSMessage) => void;

type EventHandlers = {
  onStateChange?: (state: ConnectionState) => void;
  onMessage?: MessageHandler;
  onError?: (error: Error) => void;
  onMissingSequence?: (expected: number, received: number) => void;
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
  private sequenceManager: SequenceManager;

  constructor(token: string, handlers: EventHandlers = {}) {
    this.token = token;
    const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
    const host = window.location.host;
    this.url = `${protocol}//${host}/ws/`;
    console.log('WebSocket: URL constructed', {
      protocol: window.location.protocol,
      host,
      wsUrl: this.url,
    });
    this.handlers = handlers;
    this.sequenceManager = new SequenceManager({
      onMissingSequence: handlers.onMissingSequence,
    });
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
      console.log('WebSocket: attempting to connect to', this.url);
      this.ws = new WebSocket(this.url);

      this.ws.onopen = () => {
        console.log('WebSocket: connection opened');
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

              if (message.sequence !== undefined) {
                const { messages, hasGap } = this.sequenceManager.addMessage(
                  message.sequence,
                  message,
                );
                if (hasGap) {
                  console.warn('WebSocket: detected missing sequence numbers');
                }
                for (const orderedMessage of messages) {
                  this.handlers.onMessage?.(orderedMessage as WSMessage);
                }
              } else {
                this.handlers.onMessage?.(message);
              }
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

      this.ws.onerror = (error) => {
        console.error('WebSocket: connection error', error);
        this.setState('error');
        this.handlers.onError?.(new Error('WebSocket connection error'));
      };

      this.ws.onclose = (event) => {
        console.log('WebSocket: connection closed', {
          code: event.code,
          reason: event.reason,
          wasClean: event.wasClean,
        });
        this.setState('disconnected');
        this.ws = null;
        this.sequenceManager.reset();

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

  send(message: WSMessage, addSequence = true): void {
    if (this.ws?.readyState === WebSocket.OPEN) {
      const messageToSend: WSMessage = { ...message };
      if (addSequence && message.type !== 'auth') {
        messageToSend.sequence = this.sequenceManager.getNextSequence();
      }
      this.ws.send(JSON.stringify(messageToSend));
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
