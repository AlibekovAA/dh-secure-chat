import { useState, useRef, useEffect } from 'react';
import type { ChatMessage } from './useChatSession';
import { FileMessage } from './FileMessage';
import { VoiceMessage } from './VoiceMessage';
import { MessageContextMenu } from './MessageContextMenu';
import { EmojiPicker } from './EmojiPicker';

type Props = {
  message: ChatMessage;
  myUserId: string;
  onReaction: (messageId: string, emoji: string, action: 'add' | 'remove') => void;
  onDelete?: (messageId: string, scope: 'me' | 'all') => void;
  onMediaActiveChange?: (active: boolean) => void;
  onReply?: (message: ChatMessage) => void;
};

export function MessageBubble({
  message,
  myUserId,
  onReaction,
  onDelete,
  onMediaActiveChange,
  onReply,
}: Props) {
  const [contextMenu, setContextMenu] = useState<{ x: number; y: number } | null>(null);
  const [emojiPicker, setEmojiPicker] = useState<{ x: number; y: number } | null>(null);
  const messageRef = useRef<HTMLDivElement>(null);

  const handleContextMenu = (e: React.MouseEvent) => {
    e.preventDefault();
    setContextMenu({ x: e.clientX, y: e.clientY });
  };

  const handleCopy = () => {
    if (message.text) {
      navigator.clipboard.writeText(message.text);
    }
  };

  const handleReact = () => {
    if (contextMenu) {
      setEmojiPicker({ x: contextMenu.x, y: contextMenu.y - 200 });
    }
  };

  const handleEmojiSelect = (emoji: string) => {
    const reactions = message.reactions || {};
    const emojiReactions = reactions[emoji] || [];
    const hasReacted = emojiReactions.includes(myUserId);
    onReaction(message.id, emoji, hasReacted ? 'remove' : 'add');
  };

  if (message.isDeleted) {
    return (
      <div
        className={`flex ${message.isOwn ? 'justify-end' : 'justify-start'} animate-[fadeIn_0.2s_ease-out]`}
      >
        <div
          className={`max-w-[75%] rounded-lg px-3 py-2 ${
            message.isOwn
              ? 'bg-emerald-500/10 border border-emerald-500/20 text-emerald-500/50'
              : 'bg-emerald-900/10 border border-emerald-700/20 text-emerald-500/50'
          }`}
        >
          <p className="text-sm italic">Сообщение удалено</p>
          <p className="text-[10px] text-emerald-500/40 mt-1">
            {new Date(message.timestamp).toLocaleTimeString('ru-RU', {
              hour: '2-digit',
              minute: '2-digit',
            })}
          </p>
        </div>
      </div>
    );
  }

  return (
    <>
      <div
        ref={messageRef}
        className={`flex ${message.isOwn ? 'justify-end' : 'justify-start'} animate-[fadeIn_0.2s_ease-out] group`}
      >
        <div
          className={`max-w-[75%] rounded-lg px-3 py-2 ${
            message.isOwn
              ? 'bg-emerald-500/20 border border-emerald-500/40 text-emerald-50'
              : 'bg-emerald-900/20 border border-emerald-700/40 text-emerald-100'
          }`}
          onContextMenu={handleContextMenu}
          data-message-id={message.id}
        >
          {message.replyTo && (
            <button
              type="button"
              className="mb-1 rounded border border-emerald-700/60 bg-black/60 px-2 py-1 text-[11px] text-emerald-300 text-left hover:bg-emerald-900/60 transition-colors"
              onClick={() => {
                const targetId = message.replyTo?.id;
                if (!targetId) return;
                const el = document.querySelector<HTMLElement>(
                  `[data-message-id="${targetId}"]`,
                );
                el?.scrollIntoView({ behavior: 'smooth', block: 'center' });
              }}
            >
              <span className="truncate block">
                Ответ на:{' '}
                {message.replyTo.text
                  ? message.replyTo.text
                  : message.replyTo.hasVoice
                    ? 'Голосовое сообщение'
                    : message.replyTo.hasFile
                      ? 'Файл'
                      : 'сообщение'}
              </span>
            </button>
          )}

          {message.text && (
            <p className="text-sm whitespace-pre-wrap break-words">
              {message.text}
            </p>
          )}
          {message.voice && (
            <VoiceMessage
              duration={message.voice.duration}
              blob={message.voice.blob}
              isOwn={message.isOwn}
              onPlaybackChange={onMediaActiveChange}
            />
          )}
          {message.file && !message.voice && (
            <FileMessage
              filename={message.file.filename}
              mimeType={message.file.mimeType}
              size={message.file.size}
              blob={message.file.blob}
              isOwn={message.isOwn}
              onDownloadStateChange={onMediaActiveChange}
            />
          )}

          {message.reactions && Object.keys(message.reactions).length > 0 && (
            <div className="flex flex-wrap gap-1 mt-2">
              {Object.entries(message.reactions).map(([emoji, userIds]) => {
                if (userIds.length === 0) return null;
                const hasReacted = userIds.includes(myUserId);
                return (
                  <button
                    key={emoji}
                    onClick={() =>
                      onReaction(message.id, emoji, hasReacted ? 'remove' : 'add')
                    }
                    className={`px-2 py-0.5 rounded text-xs flex items-center gap-1 transition-colors ${
                      hasReacted
                        ? 'bg-emerald-500/40 border border-emerald-500/60'
                        : 'bg-black/40 border border-emerald-500/20 hover:bg-emerald-500/20'
                    }`}
                  >
                    <span>{emoji}</span>
                    <span className="text-emerald-500/80">{userIds.length}</span>
                  </button>
                );
              })}
            </div>
          )}

          <div className="flex items-center justify-between mt-1">
            <p className="text-[10px] text-emerald-500/60">
              {new Date(message.timestamp).toLocaleTimeString('ru-RU', {
                hour: '2-digit',
                minute: '2-digit',
              })}
            </p>
          </div>
        </div>
      </div>

      {contextMenu && (
        <MessageContextMenu
          x={contextMenu.x}
          y={contextMenu.y}
          isOwn={message.isOwn}
          canEdit={false}
          onCopy={handleCopy}
          onReact={handleReact}
          onReply={onReply ? () => onReply(message) : undefined}
          onDeleteForMe={message.isOwn ? () => onDelete?.(message.id, 'me') : undefined}
          onDeleteForAll={message.isOwn ? () => onDelete?.(message.id, 'all') : undefined}
          onClose={() => setContextMenu(null)}
        />
      )}

      {emojiPicker && (
        <EmojiPicker
          x={emojiPicker.x}
          y={emojiPicker.y}
          onSelect={handleEmojiSelect}
          onClose={() => setEmojiPicker(null)}
        />
      )}
    </>
  );
}
