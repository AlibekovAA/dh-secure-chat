import { useCallback, useEffect, useRef } from 'react';
import { useVirtualizer } from '@tanstack/react-virtual';
import type { ChatMessage } from '@/modules/chat/useChatSession';
import { MessageBubble } from '@/modules/chat/MessageBubble';
import { TypingIndicator } from '@/modules/chat/TypingIndicator';
import type { PeerActivity } from '@/modules/chat/useChatSession';

type Props = {
  messages: ChatMessage[];
  myUserId: string;
  peerUsername?: string;
  peerActivity: PeerActivity | null;
  isSessionActive: boolean;
  isChatBlocked: boolean;
  onReaction: (
    messageId: string,
    emoji: string,
    action: 'add' | 'remove'
  ) => void;
  onDelete?: (messageId: string, scope: 'me' | 'all') => void;
  onEdit?: (messageId: string, newText: string) => void;
  onMarkAsRead?: (messageId: string) => void;
  onMediaActiveChange?: (active: boolean) => void;
  onReply?: (message: ChatMessage) => void;
  onEditingChange?: (editing: boolean) => void;
  onViewFile?: (
    filename: string,
    mimeType: string,
    blob: Blob,
    isProtected: boolean
  ) => void;
  scrollElementRef: React.RefObject<HTMLElement>;
};

export function MessageList({
  messages,
  myUserId,
  peerUsername,
  peerActivity,
  isSessionActive,
  isChatBlocked,
  onReaction,
  onDelete,
  onEdit,
  onMarkAsRead,
  onMediaActiveChange,
  onReply,
  onEditingChange,
  onViewFile,
  scrollElementRef,
}: Props) {
  const typingVisible =
    peerActivity === 'typing' && isSessionActive && !isChatBlocked;
  const count = messages.length + (typingVisible ? 1 : 0);

  const virtualizer = useVirtualizer({
    count,
    getScrollElement: () => scrollElementRef.current,
    estimateSize: (index) => {
      const isLastTyping = typingVisible && index === count - 1;
      if (isLastTyping) {
        return 48;
      }
      return 84;
    },
    overscan: 8,
    getItemKey: (index) => {
      const isLastTyping = typingVisible && index === count - 1;
      if (isLastTyping) {
        return 'typing';
      }
      return messages[index]?.id ?? `msg-${index}`;
    },
  });

  const isAtBottomRef = useRef(true);
  const prevMessagesLengthRef = useRef<number | null>(null);

  const jumpToMessageByIdRef = useRef<(messageId: string) => void>(() => {});
  const handleJumpToMessageById = useCallback((messageId: string) => {
    jumpToMessageByIdRef.current(messageId);
  }, []);

  useEffect(() => {
    const element = scrollElementRef.current;
    if (!element) return;

    const handleScroll = () => {
      const { scrollTop, scrollHeight, clientHeight } = element;
      isAtBottomRef.current = scrollHeight - scrollTop - clientHeight < 50;
    };

    element.addEventListener('scroll', handleScroll, { passive: true });
    handleScroll();

    return () => {
      element.removeEventListener('scroll', handleScroll);
    };
  }, [scrollElementRef]);

  const items = virtualizer.getVirtualItems();

  useEffect(() => {
    const prevLength = prevMessagesLengthRef.current;
    prevMessagesLengthRef.current = messages.length;

    if (messages.length === 0) return;

    const hasNewMessage = prevLength === null || messages.length > prevLength;
    if (!hasNewMessage || !isAtBottomRef.current) return;

    const targetIndex = typingVisible ? count - 1 : messages.length - 1;
    requestAnimationFrame(() => {
      virtualizer.scrollToIndex(targetIndex, {
        align: 'end',
        behavior: 'auto',
      });
    });
  }, [messages.length, typingVisible, count, virtualizer]);

  useEffect(() => {
    jumpToMessageByIdRef.current = (messageId: string) => {
      const targetIndex = messages.findIndex((m) => m.id === messageId);
      if (targetIndex < 0) return;
      requestAnimationFrame(() => {
        virtualizer.scrollToIndex(targetIndex, {
          align: 'center',
          behavior: 'smooth',
        });
      });
    };
  }, [messages, virtualizer]);

  return (
    <>
      <div
        style={{
          height: `${virtualizer.getTotalSize()}px`,
          width: '100%',
          position: 'relative',
        }}
      >
        {items.map((virtualItem) => {
          const isTyping = typingVisible && virtualItem.index === count - 1;

          if (isTyping) {
            return (
              <div
                key={virtualItem.key}
                data-index={virtualItem.index}
                ref={virtualizer.measureElement}
                style={{
                  position: 'absolute',
                  top: 0,
                  left: 0,
                  width: '100%',
                  transform: `translateY(${virtualItem.start}px)`,
                  paddingBottom: 12,
                }}
              >
                <TypingIndicator isVisible={true} />
              </div>
            );
          }

          const message = messages[virtualItem.index];
          if (!message) return null;

          return (
            <div
              key={virtualItem.key}
              data-index={virtualItem.index}
              ref={virtualizer.measureElement}
              style={{
                position: 'absolute',
                top: 0,
                left: 0,
                width: '100%',
                transform: `translateY(${virtualItem.start}px)`,
                paddingBottom: 12,
              }}
            >
              <MessageBubble
                message={message}
                myUserId={myUserId}
                peerUsername={peerUsername}
                onReaction={onReaction}
                onDelete={onDelete}
                onEdit={onEdit}
                onMarkAsRead={onMarkAsRead}
                onMediaActiveChange={onMediaActiveChange}
                onReply={onReply}
                onEditingChange={onEditingChange}
                onViewFile={onViewFile}
                onJumpToMessageById={handleJumpToMessageById}
              />
            </div>
          );
        })}
      </div>
    </>
  );
}
