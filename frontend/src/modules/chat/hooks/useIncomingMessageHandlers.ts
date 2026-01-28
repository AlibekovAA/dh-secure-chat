import { useCallback } from 'react';
import type {
  MessagePayload,
  ReactionPayload,
  MessageDeletePayload,
  MessageEditPayload,
  MessageReadPayload,
  WSMessage,
} from '@/shared/websocket/types';
import { decrypt } from '@/shared/crypto/encryption';
import type { SessionKey } from '@/shared/crypto/session';
import type { ChatMessage } from '@/modules/chat/useChatSession';
import { MESSAGES } from '@/shared/messages';

type UseIncomingMessageHandlersOptions = {
  sessionKeyRef: React.MutableRefObject<SessionKey | null>;
  peerId: string | null;
  messageIdCounterRef: React.MutableRefObject<number>;
  setMessages: React.Dispatch<React.SetStateAction<ChatMessage[]>>;
  setError: (error: string | null) => void;
  sendRef: React.MutableRefObject<((message: WSMessage) => void) | null>;
};

export function useIncomingMessageHandlers({
  sessionKeyRef,
  peerId,
  messageIdCounterRef,
  setMessages,
  setError,
  sendRef,
}: UseIncomingMessageHandlersOptions) {
  const handleIncomingMessage = useCallback(
    async (payload: MessagePayload) => {
      if (!sessionKeyRef.current || !peerId) return;

      try {
        const decrypted = await decrypt(
          sessionKeyRef.current,
          payload.ciphertext,
          payload.nonce
        );

        setMessages((prev) => {
          let replyTo: ChatMessage['replyTo'] | undefined;

          if (payload.reply_to_message_id) {
            const target = prev.find(
              (m) => m.id === payload.reply_to_message_id
            );
            if (target) {
              replyTo = {
                id: target.id,
                text: target.text,
                hasFile: !!target.file,
                hasVoice: !!target.voice,
                isOwn: target.isOwn,
                isDeleted: target.isDeleted,
              };
            }
          }

          const newMessage: ChatMessage = {
            id:
              payload.message_id ||
              `msg-${Date.now()}-${messageIdCounterRef.current++}`,
            text: decrypted,
            timestamp: Date.now(),
            isOwn: false,
            replyTo,
          };

          return [...prev, newMessage];
        });

        sendRef.current?.({
          type: 'ack',
          payload: {
            to: payload.from || peerId,
            message_id: payload.message_id,
          },
        });
      } catch (err) {
        setError(MESSAGES.chat.incomingMessages.errors.decryptFailed);
      }
    },
    [peerId, sessionKeyRef, messageIdCounterRef, setMessages, setError, sendRef]
  );

  const handleReaction = useCallback(
    (payload: ReactionPayload) => {
      const currentUserId = localStorage.getItem('userId') || '';
      if (payload.from && payload.from !== currentUserId) {
        const fromUserId = payload.from;
        setMessages((prev) =>
          prev.map((msg) => {
            if (msg.id === payload.message_id) {
              const reactions = msg.reactions || {};
              const emojiReactions = reactions[payload.emoji] || [];
              if (payload.action === 'add') {
                if (!emojiReactions.includes(fromUserId)) {
                  return {
                    ...msg,
                    reactions: {
                      ...reactions,
                      [payload.emoji]: [...emojiReactions, fromUserId],
                    },
                  };
                }
              } else {
                return {
                  ...msg,
                  reactions: {
                    ...reactions,
                    [payload.emoji]: emojiReactions.filter(
                      (id) => id !== fromUserId
                    ),
                  },
                };
              }
            }
            return msg;
          })
        );
      }
    },
    [setMessages]
  );

  const handleMessageDelete = useCallback(
    (payload: MessageDeletePayload) => {
      if (payload.from === peerId) {
        const scope = payload.scope ?? 'all';
        if (scope === 'all') {
          setMessages((prev) =>
            prev.map((msg) => {
              if (msg.id === payload.message_id) {
                return { ...msg, isDeleted: true, text: undefined };
              }

              if (msg.replyTo?.id === payload.message_id) {
                return {
                  ...msg,
                  replyTo: {
                    ...msg.replyTo,
                    isDeleted: true,
                    text: undefined,
                    hasFile: false,
                    hasVoice: false,
                  },
                };
              }

              return msg;
            })
          );
        }
      }
    },
    [peerId, setMessages]
  );

  const handleMessageEdit = useCallback(
    async (payload: MessageEditPayload) => {
      if (!sessionKeyRef.current || payload.from !== peerId) return;

      try {
        const decrypted = await decrypt(
          sessionKeyRef.current,
          payload.ciphertext,
          payload.nonce
        );

        setMessages((prev) =>
          prev.map((msg) => {
            if (msg.id === payload.message_id) {
              return {
                ...msg,
                text: decrypted,
                isEdited: true,
              };
            }

            if (msg.replyTo?.id === payload.message_id) {
              return {
                ...msg,
                replyTo: {
                  ...msg.replyTo,
                  text: decrypted,
                },
              };
            }

            return msg;
          })
        );
      } catch (err) {
        setError(MESSAGES.chat.incomingMessages.errors.decryptEditedFailed);
      }
    },
    [peerId, sessionKeyRef, setMessages, setError]
  );

  const handleMessageRead = useCallback(
    (payload: MessageReadPayload) => {
      if (payload.from === peerId) {
        setMessages((prev) =>
          prev.map((msg) => {
            if (msg.id === payload.message_id && msg.isOwn) {
              return { ...msg, deliveryStatus: 'read' as const };
            }
            return msg;
          })
        );
      }
    },
    [peerId, setMessages]
  );

  return {
    handleIncomingMessage,
    handleReaction,
    handleMessageDelete,
    handleMessageEdit,
    handleMessageRead,
  };
}
