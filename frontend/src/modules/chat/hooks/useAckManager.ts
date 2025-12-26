import { useCallback } from 'react';
import type {
  WSMessage,
  EphemeralKeyPayload,
} from '../../../shared/websocket/types';

const ACK_TIMEOUT = 5000;
const MAX_RETRIES = 3;
const RETRY_DELAY = 1000;

type PendingAck = {
  message: WSMessage;
  timeout: number;
  retries: number;
  timestamp: number;
};

type UseAckManagerOptions = {
  sendRef: React.MutableRefObject<((message: WSMessage) => void) | null>;
  pendingAcksRef: React.MutableRefObject<Map<string, PendingAck>>;
  onMaxRetries: () => void;
};

export function useAckManager({
  sendRef,
  pendingAcksRef,
  onMaxRetries,
}: UseAckManagerOptions) {
  const handleAck = useCallback((messageId: string) => {
    const pending = pendingAcksRef.current.get(messageId);
    if (pending) {
      clearTimeout(pending.timeout);
      pendingAcksRef.current.delete(messageId);
    }
  }, []);

  const scheduleRetry = useCallback(
    (messageId: string) => {
      const pending = pendingAcksRef.current.get(messageId);
      if (!pending) return;

      pending.retries++;
      if (pending.retries >= MAX_RETRIES) {
        pendingAcksRef.current.delete(messageId);
        onMaxRetries();
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
    },
    [sendRef, onMaxRetries],
  );

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
    [sendRef, scheduleRetry],
  );

  const clearPendingAcks = useCallback(() => {
    for (const [, pending] of pendingAcksRef.current) {
      clearTimeout(pending.timeout);
    }
    pendingAcksRef.current.clear();
  }, []);

  return {
    handleAck,
    sendWithAck,
    clearPendingAcks,
  };
}
