import { useCallback } from 'react';
import type { WSMessage, EphemeralKeyPayload } from '@/shared/websocket/types';
import {
  ACK_TIMEOUT_MS,
  ACK_MAX_RETRIES,
  ACK_RETRY_DELAY_MS,
} from '@/shared/constants';

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
  const handleAck = useCallback(
    (messageId: string) => {
      const pending = pendingAcksRef.current.get(messageId);
      if (pending) {
        clearTimeout(pending.timeout);
        pendingAcksRef.current.delete(messageId);
      }
    },
    [pendingAcksRef]
  );

  const scheduleRetry = useCallback(
    (messageId: string) => {
      const pending = pendingAcksRef.current.get(messageId);
      if (!pending) return;

      pending.retries++;
      if (pending.retries >= ACK_MAX_RETRIES) {
        pendingAcksRef.current.delete(messageId);
        onMaxRetries();
        return;
      }

      const retryTimeout = setTimeout(() => {
        sendRef.current?.(pending.message);
        const ackTimeout = setTimeout(() => {
          scheduleRetry(messageId);
        }, ACK_TIMEOUT_MS) as unknown as number;
        pending.timeout = ackTimeout;
      }, ACK_RETRY_DELAY_MS) as unknown as number;

      pending.timeout = retryTimeout;
    },
    [sendRef, onMaxRetries, pendingAcksRef]
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
      }, ACK_TIMEOUT_MS) as unknown as number;

      pendingAcksRef.current.set(messageId, {
        message,
        timeout,
        retries: 0,
        timestamp: Date.now(),
      });

      sendRef.current?.(message);
    },
    [sendRef, scheduleRetry, pendingAcksRef]
  );

  const clearPendingAcks = useCallback(() => {
    for (const [, pending] of pendingAcksRef.current) {
      clearTimeout(pending.timeout);
    }
    pendingAcksRef.current.clear();
  }, [pendingAcksRef]);

  return {
    handleAck,
    sendWithAck,
    clearPendingAcks,
  };
}
