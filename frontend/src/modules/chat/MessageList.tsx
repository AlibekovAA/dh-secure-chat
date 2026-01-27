import { useRef } from 'react';
import type { ChatMessage } from '@/modules/chat/useChatSession';
import { MessageBubble } from '@/modules/chat/MessageBubble';
import { TypingIndicator } from '@/modules/chat/TypingIndicator';

type Props = {
  messages: ChatMessage[];
  myUserId: string;
  peerUsername?: string;
  isPeerTyping: boolean;
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
  messagesEndRef?: React.RefObject<HTMLDivElement>;
};

export function MessageList({
  messages,
  myUserId,
  peerUsername,
  isPeerTyping,
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
  messagesEndRef: externalMessagesEndRef,
}: Props) {
  const internalMessagesEndRef = useRef<HTMLDivElement>(null);
  const messagesEndRef = externalMessagesEndRef || internalMessagesEndRef;

  return (
    <>
      {messages.map((message) => (
        <MessageBubble
          key={message.id}
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
        />
      ))}

      <TypingIndicator
        isVisible={isPeerTyping && isSessionActive && !isChatBlocked}
      />

      <div ref={messagesEndRef} />
    </>
  );
}
