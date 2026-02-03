import { useState, useRef, useEffect, memo } from 'react';
import type { ChatMessage } from '@/modules/chat/useChatSession';
import { FileMessage } from '@/modules/chat/FileMessage';
import { VoiceMessage } from '@/modules/chat/VoiceMessage';
import { VideoCircle } from '@/modules/chat/VideoCircle';
import { MessageContextMenu } from '@/modules/chat/MessageContextMenu';
import { EmojiPicker } from '@/modules/chat/EmojiPicker';
import {
  MESSAGE_READ_INTERSECTION_THRESHOLD,
  EDIT_TIMEOUT_MS,
  MESSAGE_MAX_WIDTH_PERCENT,
  TEXTAREA_MAX_ROWS,
} from '@/shared/constants';
import { formatTime } from '@/shared/utils/format';
import { MESSAGES } from '@/shared/messages';

type Props = {
  message: ChatMessage;
  myUserId: string;
  peerUsername?: string;
  onJumpToMessageById?: (messageId: string) => void;
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
};

type AnchorRect = {
  left: number;
  right: number;
  top: number;
  bottom: number;
  width: number;
  height: number;
};

function MessageBubbleComponent({
  message,
  myUserId,
  peerUsername,
  onJumpToMessageById,
  onReaction,
  onDelete,
  onEdit,
  onMarkAsRead,
  onMediaActiveChange,
  onReply,
  onEditingChange,
  onViewFile,
}: Props) {
  const [contextMenuAnchor, setContextMenuAnchor] = useState<AnchorRect | null>(
    null
  );
  const [emojiPickerAnchor, setEmojiPickerAnchor] = useState<AnchorRect | null>(
    null
  );
  const [isEditing, setIsEditing] = useState(false);
  const [editText, setEditText] = useState('');
  const messageRef = useRef<HTMLDivElement>(null);
  const editInputRef = useRef<HTMLTextAreaElement>(null);
  const [reactionsPage, setReactionsPage] = useState(0);

  const handleContextMenu = (e: React.MouseEvent) => {
    e.preventDefault();
    const bubbleRect = e.currentTarget.getBoundingClientRect();

    setContextMenuAnchor({
      left: bubbleRect.left,
      right: bubbleRect.right,
      top: bubbleRect.top,
      bottom: bubbleRect.bottom,
      width: bubbleRect.width,
      height: bubbleRect.height,
    });
  };

  const handleCopy = () => {
    if (message.text) {
      navigator.clipboard.writeText(message.text);
    }
  };

  const handleReact = () => {
    if (contextMenuAnchor) {
      setEmojiPickerAnchor(contextMenuAnchor);
      setContextMenuAnchor(null);
    }
  };

  const handleEmojiSelect = (emoji: string) => {
    const reactions = message.reactions || {};
    const emojiReactions = reactions[emoji] || [];
    const hasReacted = emojiReactions.includes(myUserId);
    onReaction(message.id, emoji, hasReacted ? 'remove' : 'add');
  };

  const handleEdit = () => {
    if (!message.text || !onEdit) return;
    setEditText(message.text);
    setIsEditing(true);
    onEditingChange?.(true);
    setContextMenuAnchor(null);
  };

  const handleSaveEdit = () => {
    if (!onEdit || !editText.trim() || editText.trim() === message.text) {
      setIsEditing(false);
      onEditingChange?.(false);
      return;
    }
    onEdit(message.id, editText.trim());
    setIsEditing(false);
    onEditingChange?.(false);
  };

  const handleCancelEdit = () => {
    setIsEditing(false);
    onEditingChange?.(false);
    setEditText('');
  };

  useEffect(() => {
    if (isEditing && editInputRef.current) {
      editInputRef.current.focus();
      editInputRef.current.select();
    }
  }, [isEditing]);

  useEffect(() => {
    if (!message.reactions) {
      setReactionsPage(0);
      return;
    }
    const entries = Object.entries(message.reactions).filter(
      ([, userIds]) => userIds.length > 0
    );
    const maxPage =
      entries.length > 0 ? Math.max(0, Math.ceil(entries.length / 2) - 1) : 0;
    setReactionsPage((prev) => Math.min(prev, maxPage));
  }, [message.reactions, message.id]);

  useEffect(() => {
    if (message.isOwn || !onMarkAsRead || !messageRef.current) return;

    const observer = new IntersectionObserver(
      (entries) => {
        entries.forEach((entry) => {
          if (entry.isIntersecting) {
            onMarkAsRead(message.id);
            observer.disconnect();
          }
        });
      },
      { threshold: MESSAGE_READ_INTERSECTION_THRESHOLD }
    );

    observer.observe(messageRef.current);

    return () => {
      observer.disconnect();
    };
  }, [message.id, message.isOwn, onMarkAsRead]);

  if (message.isDeleted) {
    return (
      <div
        className={`flex ${message.isOwn ? 'justify-end' : 'justify-start'} animate-[fadeIn_0.2s_ease-out]`}
        style={{ willChange: 'opacity' }}
      >
        <div
          className={`rounded-lg px-3 py-2.5 border-dashed ${
            message.isOwn
              ? 'bg-gradient-to-br from-emerald-500/5 to-emerald-500/10 border border-emerald-500/30 text-emerald-400/60'
              : 'bg-gradient-to-br from-emerald-900/5 to-emerald-900/10 border border-emerald-700/30 text-emerald-500/60'
          }`}
          style={{ maxWidth: `${MESSAGE_MAX_WIDTH_PERCENT}%` }}
        >
          <div className="flex items-center gap-2">
            <svg
              className="w-3.5 h-3.5 text-emerald-500/50"
              fill="none"
              stroke="currentColor"
              viewBox="0 0 24 24"
            >
              <path
                strokeLinecap="round"
                strokeLinejoin="round"
                strokeWidth={2}
                d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16"
              />
            </svg>
            <p className="text-sm italic font-medium">
              {MESSAGES.chat.messageBubble.deleted}
            </p>
          </div>
          <p className="text-xs text-emerald-500/50 mt-1.5 leading-relaxed font-mono">
            {formatTime(message.timestamp)}
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
        style={{ willChange: 'opacity' }}
      >
        <div
          className={`max-w-[75%] rounded-lg overflow-hidden smooth-transition ${
            message.isOwn
              ? 'bg-emerald-500/20 border border-emerald-500/40 text-emerald-50 hover:bg-emerald-500/25 hover:border-emerald-500/50'
              : 'bg-emerald-900/20 border border-emerald-700/40 text-emerald-100 hover:bg-emerald-900/25 hover:border-emerald-700/50'
          } ${contextMenuAnchor ? 'relative z-[120] scale-[1.02] shadow-2xl shadow-emerald-900/40' : ''}`}
          style={{ willChange: 'background-color, border-color' }}
          onContextMenu={handleContextMenu}
          data-message-id={message.id}
        >
          {message.replyTo && (
            <button
              type="button"
              className="w-full text-left border-l-4 border-emerald-400/60 bg-emerald-900/10 hover:bg-emerald-900/20 transition-colors"
              onClick={() => {
                const targetId = message.replyTo?.id;
                if (!targetId) return;
                if (onJumpToMessageById) {
                  onJumpToMessageById(targetId);
                  return;
                }
                const el = document.querySelector<HTMLElement>(
                  `[data-message-id="${targetId}"]`
                );
                el?.scrollIntoView({ behavior: 'smooth', block: 'center' });
              }}
            >
              <div className="pl-3 pr-3 py-1.5">
                <p className="text-xs font-medium text-emerald-400/90 mb-0.5">
                  {message.replyTo.isOwn
                    ? MESSAGES.chat.messageBubble.you
                    : peerUsername || MESSAGES.chat.messageBubble.peerFallback}
                </p>
                <p className="text-xs text-emerald-200/70 line-clamp-2 break-words">
                  {message.replyTo.isDeleted
                    ? MESSAGES.chat.messageBubble.replyPreview.deleted
                    : message.replyTo.text
                      ? message.replyTo.text
                      : message.replyTo.hasVoice
                        ? MESSAGES.chat.messageBubble.replyPreview.voice
                        : message.replyTo.hasVideo
                          ? MESSAGES.chat.messageBubble.replyPreview.video
                          : message.replyTo.hasFile
                            ? MESSAGES.chat.messageBubble.replyPreview.file
                            : MESSAGES.chat.messageBubble.replyPreview.message}
                </p>
              </div>
            </button>
          )}

          <div className="px-3 pb-2 pt-2">
            {isEditing ? (
              <div className="space-y-2">
                <textarea
                  ref={editInputRef}
                  data-edit-input
                  value={editText}
                  onChange={(e) => setEditText(e.target.value)}
                  onKeyDown={(e) => {
                    if (e.key === 'Enter' && !e.shiftKey) {
                      e.preventDefault();
                      handleSaveEdit();
                    } else if (e.key === 'Escape') {
                      handleCancelEdit();
                    }
                  }}
                  className="w-full rounded-md bg-black/60 border border-emerald-500/60 px-3 py-2 text-sm text-emerald-50 outline-none focus:ring-2 focus:ring-emerald-500 resize-none"
                  rows={Math.min(
                    editText.split('\n').length,
                    TEXTAREA_MAX_ROWS
                  )}
                />
                <div className="flex items-center gap-2">
                  <button
                    onClick={handleSaveEdit}
                    className="px-3 py-1 text-xs rounded-md bg-emerald-500 hover:bg-emerald-400 text-black transition-colors"
                  >
                    {MESSAGES.chat.messageBubble.actions.save}
                  </button>
                  <button
                    onClick={handleCancelEdit}
                    className="px-3 py-1 text-xs rounded-md bg-emerald-900/40 hover:bg-emerald-900/60 text-emerald-300 transition-colors"
                  >
                    {MESSAGES.chat.messageBubble.actions.cancel}
                  </button>
                </div>
              </div>
            ) : (
              <>
                {message.text && (
                  <p className="text-[0.95rem] whitespace-pre-wrap break-words leading-relaxed text-emerald-50/95">
                    {message.text}
                    {message.isEdited && (
                      <span className="ml-2 text-[10px] text-emerald-500/60 italic">
                        {MESSAGES.chat.messageBubble.edited}
                      </span>
                    )}
                  </p>
                )}
              </>
            )}
            {message.voice && (
              <VoiceMessage
                duration={message.voice.duration}
                blob={message.voice.blob}
                isOwn={message.isOwn}
                onPlaybackChange={onMediaActiveChange}
                onMarkAsRead={
                  !message.isOwn && onMarkAsRead
                    ? () => {
                        onMarkAsRead(message.id);
                      }
                    : undefined
                }
              />
            )}
            {message.video && (
              <VideoCircle
                blob={message.video.blob!}
                filename={message.video.filename}
                fileId={message.id}
                isOwn={message.isOwn}
              />
            )}
            {message.file && !message.voice && !message.video && (
              <FileMessage
                filename={message.file.filename}
                mimeType={message.file.mimeType}
                size={message.file.size}
                blob={message.file.blob}
                isOwn={message.isOwn}
                accessMode={message.file.accessMode}
                onDownloadStateChange={onMediaActiveChange}
                onView={
                  message.file.blob && onViewFile
                    ? () => {
                        if (!message.isOwn && onMarkAsRead) {
                          onMarkAsRead(message.id);
                        }
                        onViewFile(
                          message.file!.filename,
                          message.file!.mimeType,
                          message.file!.blob!,
                          !message.isOwn &&
                            message.file!.accessMode === 'view_only'
                        );
                      }
                    : undefined
                }
              />
            )}

            {(() => {
              const reactionEntries = message.reactions
                ? Object.entries(message.reactions).filter(
                    ([, userIds]) => userIds.length > 0
                  )
                : [];
              const hasReactions = reactionEntries.length > 0;
              const pageSize = 2;
              const totalPages = Math.max(
                1,
                Math.ceil(reactionEntries.length / pageSize)
              );
              const currentPage = Math.min(reactionsPage, totalPages - 1);
              const startIndex = currentPage * pageSize;
              const visibleReactions = reactionEntries.slice(
                startIndex,
                startIndex + pageSize
              );

              if (!hasReactions) {
                return null;
              }

              return (
                <div
                  className={`mt-1.5 mb-0.5 flex items-center gap-1 ${
                    message.isOwn ? 'justify-end' : 'justify-start'
                  }`}
                >
                  {totalPages > 1 && (
                    <button
                      type="button"
                      onClick={() =>
                        setReactionsPage((prev) =>
                          prev > 0 ? prev - 1 : totalPages - 1
                        )
                      }
                      className="w-6 h-6 flex items-center justify-center rounded-full border border-emerald-700/60 bg-black/40 text-[10px] text-emerald-300 hover:bg-emerald-900/60 smooth-transition"
                    >
                      ←
                    </button>
                  )}
                  <div className="flex flex-wrap gap-1">
                    {visibleReactions.map(([emoji, userIds]) => {
                      const hasReacted = userIds.includes(myUserId);
                      return (
                        <button
                          key={emoji}
                          onClick={() =>
                            onReaction(
                              message.id,
                              emoji,
                              hasReacted ? 'remove' : 'add'
                            )
                          }
                          className={`group/reaction inline-flex items-center gap-1 rounded-full px-2.5 py-1 text-[11px] font-medium shadow-sm transition-all duration-150 animate-[fadeIn_0.18s_ease-out] ${
                            hasReacted
                              ? message.isOwn
                                ? 'bg-gradient-to-br from-emerald-400 to-emerald-500 text-black shadow-emerald-900/40 ring-1 ring-emerald-300/80'
                                : 'bg-gradient-to-br from-emerald-500/80 to-emerald-400/80 text-black shadow-emerald-900/30 ring-1 ring-emerald-300/70'
                              : message.isOwn
                                ? 'bg-emerald-500/15 border border-emerald-400/50 text-emerald-50/90 hover:bg-emerald-500/20 shadow-emerald-900/20'
                                : 'bg-emerald-900/40 border border-emerald-600/40 text-emerald-50/90 hover:bg-emerald-900/55 shadow-emerald-900/30'
                          } hover:scale-[1.04] active:scale-[0.97]`}
                        >
                          <span className="text-lg leading-none">
                            {emoji}
                          </span>
                          <span className="text-[10px] font-semibold text-emerald-100/90">
                            {userIds.length}
                          </span>
                        </button>
                      );
                    })}
                  </div>
                  {totalPages > 1 && (
                    <button
                      type="button"
                      onClick={() =>
                        setReactionsPage((prev) =>
                          prev < totalPages - 1 ? prev + 1 : 0
                        )
                      }
                      className="w-6 h-6 flex items-center justify-center rounded-full border border-emerald-700/60 bg-black/40 text-[10px] text-emerald-300 hover:bg-emerald-900/60 smooth-transition"
                    >
                      →
                    </button>
                  )}
                </div>
              );
            })()}

            <div className="flex items-center justify-between border-t border-emerald-500/10 pt-1.5 mt-2.5">
              <p className="text-xs text-emerald-400/70 leading-relaxed font-medium">
                {formatTime(message.timestamp)}
              </p>
              {message.isOwn && message.deliveryStatus && (
                <div className="flex items-center gap-1">
                  {message.deliveryStatus === 'sending' && (
                    <svg
                      className="w-3.5 h-3.5 text-emerald-500/50"
                      fill="currentColor"
                      viewBox="0 0 20 20"
                    >
                      <path
                        fillRule="evenodd"
                        d="M16.707 5.293a1 1 0 010 1.414l-8 8a1 1 0 01-1.414 0l-4-4a1 1 0 011.414-1.414L8 12.586l7.293-7.293a1 1 0 011.414 0z"
                        clipRule="evenodd"
                      />
                    </svg>
                  )}
                  {message.deliveryStatus === 'delivered' && (
                    <svg
                      className="w-4 h-3.5 text-emerald-500/70"
                      fill="currentColor"
                      viewBox="0 0 20 20"
                    >
                      <path
                        fillRule="evenodd"
                        d="M16.707 5.293a1 1 0 010 1.414l-8 8a1 1 0 01-1.414 0l-4-4a1 1 0 011.414-1.414L8 12.586l7.293-7.293a1 1 0 011.414 0z"
                        clipRule="evenodd"
                      />
                      <path
                        fillRule="evenodd"
                        d="M16.707 7.293a1 1 0 010 1.414l-8 8a1 1 0 01-1.414 0l-4-4a1 1 0 011.414-1.414L8 14.586l7.293-7.293a1 1 0 011.414 0z"
                        clipRule="evenodd"
                      />
                    </svg>
                  )}
                  {message.deliveryStatus === 'read' && (
                    <svg
                      className="w-4 h-3.5 text-blue-400"
                      fill="currentColor"
                      viewBox="0 0 20 20"
                    >
                      <path
                        fillRule="evenodd"
                        d="M16.707 5.293a1 1 0 010 1.414l-8 8a1 1 0 01-1.414 0l-4-4a1 1 0 011.414-1.414L8 12.586l7.293-7.293a1 1 0 011.414 0z"
                        clipRule="evenodd"
                      />
                      <path
                        fillRule="evenodd"
                        d="M16.707 7.293a1 1 0 010 1.414l-8 8a1 1 0 01-1.414 0l-4-4a1 1 0 011.414-1.414L8 14.586l7.293-7.293a1 1 0 011.414 0z"
                        clipRule="evenodd"
                      />
                    </svg>
                  )}
                </div>
              )}
            </div>
          </div>
        </div>
      </div>

      {contextMenuAnchor && (
        <MessageContextMenu
          anchorRect={contextMenuAnchor}
          isOwn={message.isOwn}
          canEdit={
            message.isOwn &&
            !!message.text &&
            !message.isDeleted &&
            Date.now() - message.timestamp <= EDIT_TIMEOUT_MS
          }
          onCopy={handleCopy}
          onReact={handleReact}
          onReply={onReply ? () => onReply(message) : undefined}
          onEdit={
            message.isOwn && !!message.text && !message.isDeleted
              ? handleEdit
              : undefined
          }
          onDeleteForMe={
            message.isOwn ? () => onDelete?.(message.id, 'me') : undefined
          }
          onDeleteForAll={
            message.isOwn ? () => onDelete?.(message.id, 'all') : undefined
          }
          onClose={() => setContextMenuAnchor(null)}
        />
      )}

      {emojiPickerAnchor && (
        <EmojiPicker
          anchorRect={emojiPickerAnchor}
          isOwn={message.isOwn}
          onSelect={handleEmojiSelect}
          onClose={() => setEmojiPickerAnchor(null)}
        />
      )}
    </>
  );
}

function reactionsEqual(
  a?: Record<string, string[]>,
  b?: Record<string, string[]>
) {
  if (a === b) return true;
  if (!a || !b) return !a && !b;
  const aKeys = Object.keys(a);
  const bKeys = Object.keys(b);
  if (aKeys.length !== bKeys.length) return false;
  for (const key of aKeys) {
    const av = a[key] ?? [];
    const bv = b[key] ?? [];
    if (av === bv) continue;
    if (av.length !== bv.length) return false;
    for (let i = 0; i < av.length; i++) {
      if (av[i] !== bv[i]) return false;
    }
  }
  return true;
}

export const MessageBubble = memo(
  MessageBubbleComponent,
  (prevProps, nextProps) => {
    return (
      prevProps.message.id === nextProps.message.id &&
      prevProps.message.text === nextProps.message.text &&
      prevProps.message.isDeleted === nextProps.message.isDeleted &&
      prevProps.message.isEdited === nextProps.message.isEdited &&
      prevProps.message.timestamp === nextProps.message.timestamp &&
      reactionsEqual(
        prevProps.message.reactions,
        nextProps.message.reactions
      ) &&
      prevProps.message.file?.filename === nextProps.message.file?.filename &&
      prevProps.message.file?.mimeType === nextProps.message.file?.mimeType &&
      prevProps.message.file?.size === nextProps.message.file?.size &&
      prevProps.message.file?.accessMode ===
        nextProps.message.file?.accessMode &&
      prevProps.message.file?.blob === nextProps.message.file?.blob &&
      prevProps.message.voice?.filename === nextProps.message.voice?.filename &&
      prevProps.message.voice?.mimeType === nextProps.message.voice?.mimeType &&
      prevProps.message.voice?.size === nextProps.message.voice?.size &&
      prevProps.message.voice?.duration === nextProps.message.voice?.duration &&
      prevProps.message.voice?.blob === nextProps.message.voice?.blob &&
      prevProps.message.video?.filename === nextProps.message.video?.filename &&
      prevProps.message.video?.mimeType === nextProps.message.video?.mimeType &&
      prevProps.message.video?.size === nextProps.message.video?.size &&
      prevProps.message.video?.duration === nextProps.message.video?.duration &&
      prevProps.message.video?.blob === nextProps.message.video?.blob &&
      prevProps.message.replyTo?.id === nextProps.message.replyTo?.id &&
      prevProps.message.replyTo?.isDeleted ===
        nextProps.message.replyTo?.isDeleted &&
      prevProps.message.replyTo?.text === nextProps.message.replyTo?.text &&
      prevProps.message.replyTo?.hasFile ===
        nextProps.message.replyTo?.hasFile &&
      prevProps.message.replyTo?.hasVoice ===
        nextProps.message.replyTo?.hasVoice &&
      prevProps.message.replyTo?.hasVideo ===
        nextProps.message.replyTo?.hasVideo &&
      prevProps.message.replyTo?.isOwn === nextProps.message.replyTo?.isOwn &&
      prevProps.message.deliveryStatus === nextProps.message.deliveryStatus &&
      prevProps.myUserId === nextProps.myUserId &&
      prevProps.peerUsername === nextProps.peerUsername &&
      prevProps.onJumpToMessageById === nextProps.onJumpToMessageById
    );
  }
);
