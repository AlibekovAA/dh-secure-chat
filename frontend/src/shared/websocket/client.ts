import type { WSMessage, ConnectionState } from '@/shared/websocket/types';
import { SequenceManager } from '@/shared/websocket/sequence-manager';
import {
  MAX_RECONNECT_DELAY_MS,
  WEBSOCKET_MAX_RECONNECT_ATTEMPTS,
  WEBSOCKET_BASE_DELAY_MS,
} from '@/shared/constants';
import { MESSAGES } from '@/shared/messages';

type MessageHandler = (message: WSMessage) => void;

type EventHandlers = {
  onStateChange?: (state: ConnectionState) => void;
  onMessage?: MessageHandler;
  onError?: (error: Error) => void;
  onTokenExpired?: () => Promise<string | null>;
};

export class WebSocketClient {
  private ws: WebSocket | null = null;
  private token: string;
  private url: string;
  private handlers: EventHandlers;
  private reconnectAttempts = 0;
  private maxReconnectAttempts = WEBSOCKET_MAX_RECONNECT_ATTEMPTS;
  private baseDelay = WEBSOCKET_BASE_DELAY_MS;
  private reconnectTimer: number | null = null;
  private state: ConnectionState = 'disconnected';
  private sequenceManager: SequenceManager;
  private isRefreshingToken = false;

  constructor(token: string, handlers: EventHandlers = {}) {
    this.token = token;
    const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
    const host = window.location.host;
    this.url = `${protocol}//${host}/ws/`;
    this.handlers = handlers;
    this.sequenceManager = new SequenceManager();
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
          })
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
                  this.isRefreshingToken = false;
                  continue;
                }
                if (payload.error) {
                  const errorMessage = payload.error.toLowerCase();
                  const isTokenError =
                    errorMessage.includes('token') ||
                    errorMessage.includes('invalid') ||
                    errorMessage.includes('expired') ||
                    errorMessage.includes('unauthorized');

                  if (
                    isTokenError &&
                    this.handlers.onTokenExpired &&
                    !this.isRefreshingToken
                  ) {
                    this.isRefreshingToken = true;
                    this.ws?.close();
                    (async () => {
                      try {
                        const newToken = await this.handlers.onTokenExpired!();
                        if (newToken) {
                          this.token = newToken;
                          this.reconnectAttempts = 0;
                          setTimeout(() => {
                            this.connect();
                          }, 500);
                          return;
                        }
                        this.isRefreshingToken = false;
                        this.handlers.onError?.(
                          new Error(
                            MESSAGES.common.websocket.tokenRefreshFailed
                          )
                        );
                      } catch (err) {
                        this.isRefreshingToken = false;
                        this.handlers.onError?.(
                          new Error(MESSAGES.common.websocket.tokenRefreshError)
                        );
                      }
                    })();
                    return;
                  }

                  this.handlers.onError?.(
                    new Error(
                      isTokenError
                        ? MESSAGES.common.websocket.tokenExpiredRefreshing
                        : payload.error
                    )
                  );
                  this.ws?.close();
                  return;
                }
              }

              if (this.state === 'connecting') {
                this.setState('connected');
                this.reconnectAttempts = 0;
              }

              if (message.sequence !== undefined) {
                const { messages } = this.sequenceManager.addMessage(
                  message.sequence,
                  message
                );
                for (const orderedMessage of messages) {
                  this.handlers.onMessage?.(orderedMessage as WSMessage);
                }
              } else {
                this.handlers.onMessage?.(message);
              }
            } catch (_parseErr) {
              void _parseErr;
            }
          }
        } catch (err) {
          this.handlers.onError?.(
            new Error(MESSAGES.common.websocket.messageHandlingError)
          );
        }
      };

      this.ws.onerror = () => {
        this.setState('error');
        this.handlers.onError?.(
          new Error(MESSAGES.common.websocket.connectionError)
        );
      };

      this.ws.onclose = () => {
        this.setState('disconnected');
        this.ws = null;
        this.sequenceManager.reset();

        if (this.isRefreshingToken) {
          return;
        }

        if (this.reconnectAttempts < this.maxReconnectAttempts) {
          this.scheduleReconnect();
        } else {
          this.handlers.onError?.(
            new Error(MESSAGES.common.websocket.failedToConnect)
          );
        }
      };
    } catch (err) {
      this.setState('error');
      this.handlers.onError?.(
        err instanceof Error
          ? err
          : new Error(MESSAGES.common.websocket.failedToEstablishConnection)
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

  updateToken(newToken: string): void {
    this.token = newToken;
    this.isRefreshingToken = false;
  }

  send(message: WSMessage, addSequence = true): void {
    if (this.ws?.readyState === WebSocket.OPEN) {
      const messageToSend: WSMessage = { ...message };
      if (addSequence && message.type !== 'auth') {
        messageToSend.sequence = this.sequenceManager.getNextSequence();
      }
      this.ws.send(JSON.stringify(messageToSend));
    } else {
      this.handlers.onError?.(
        new Error(MESSAGES.common.websocket.notConnected)
      );
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
      MAX_RECONNECT_DELAY_MS
    );

    this.reconnectTimer = setTimeout(() => {
      this.reconnectTimer = null;
      this.connect();
    }, delay) as unknown as number;
  }
}
