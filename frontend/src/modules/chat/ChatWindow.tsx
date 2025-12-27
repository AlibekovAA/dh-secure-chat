import { useCallback, useEffect, useMemo, useRef, useState } from 'react';
import { useChatSession, type ChatMessage } from './useChatSession';
import type { UserSummary } from './api';
import { FingerprintVerificationModal } from './FingerprintVerificationModal';
import { VoiceRecorder } from './VoiceRecorder';
import { MessageBubble } from './MessageBubble';
import { TypingIndicator } from './TypingIndicator';
import { FileAccessDialog, type FileAccessMode } from './FileAccessDialog';
import { FileViewerModal } from './FileViewerModal';
import { getFingerprint } from './api';
import {
  getVerifiedPeerFingerprint,
  isPeerVerified,
  normalizeFingerprint,
} from '../../shared/crypto/fingerprint';
import { useToast } from '../../shared/ui/ToastProvider';
import { MAX_FILE_SIZE, MAX_MESSAGE_LENGTH } from './constants';

type Props = {
  token: string;
  peer: UserSummary;
  myUserId: string;
  onClose(): void;
  onTokenExpired?: () => Promise<string | null>;
};

export function ChatWindow({ token, peer, myUserId, onClose, onTokenExpired }: Props) {
  const [messageText, setMessageText] = useState('');
  const [isSending, setIsSending] = useState(false);
  const [isSendingFile, setIsSendingFile] = useState(false);
  const fileInputRef = useRef<HTMLInputElement>(null);
  const [showFingerprintModal, setShowFingerprintModal] = useState(false);
  const [myFingerprint, setMyFingerprint] = useState<string | null>(null);
  const [peerFingerprint, setPeerFingerprint] = useState<string | null>(null);
  const [fingerprintWarning, setFingerprintWarning] = useState(false);
  const [isChatBlocked, setIsChatBlocked] = useState(false);
  const [isFingerprintVerified, setIsFingerprintVerified] = useState(false);
  const [isLoadingFingerprint, setIsLoadingFingerprint] = useState(true);
  const [pendingFile, setPendingFile] = useState<File | null>(null);
  const [showAccessDialog, setShowAccessDialog] = useState(false);
  const messagesEndRef = useRef<HTMLDivElement>(null);
  const inputRef = useRef<HTMLTextAreaElement>(null);
  const typingTimeoutRef = useRef<number | null>(null);
  const { showToast } = useToast();
  const [activeMediaCount, setActiveMediaCount] = useState(0);
  const isMediaActive = activeMediaCount > 0;
  const [hasShownMaxLengthToast, setHasShownMaxLengthToast] = useState(false);
  const [replyTo, setReplyTo] = useState<ChatMessage | null>(null);
  const [viewerFile, setViewerFile] = useState<{ filename: string; mimeType: string; blob: Blob; isProtected: boolean } | null>(null);
  const [isEditingMessage, setIsEditingMessage] = useState(false);

  const {
    state,
    messages,
    error,
    sendMessage,
    sendFile,
    sendVoice,
    sendTyping,
    sendReaction,
    deleteMessage,
    editMessage,
    markMessageAsRead,
    isPeerTyping,
    isSessionActive,
  } = useChatSession({
    token,
    peerId: peer.id,
    enabled: isFingerprintVerified,
    onTokenExpired,
  });

  const scrollToBottom = useCallback(() => {
    messagesEndRef.current?.scrollIntoView({ behavior: 'smooth' });
  }, []);

  useEffect(() => {
    if (messages.length > 0) {
      const timer = setTimeout(scrollToBottom, 50);
      return () => clearTimeout(timer);
    }
  }, [messages.length, scrollToBottom]);

  useEffect(() => {
    if (isPeerTyping && isSessionActive && !isChatBlocked) {
      const timer = setTimeout(scrollToBottom, 100);
      return () => clearTimeout(timer);
    }
  }, [isPeerTyping, isSessionActive, isChatBlocked, scrollToBottom]);

  useEffect(() => {
    if (
      inputRef.current &&
      !isChatBlocked &&
      isSessionActive &&
      !isSending &&
      !isMediaActive &&
      !showAccessDialog &&
      !viewerFile &&
      !isEditingMessage
    ) {
      const timer = setTimeout(() => {
        const activeElement = document.activeElement;
        const isEditing = document.querySelector('textarea[data-edit-input]') !== null;
        if (!isEditing && activeElement?.tagName !== 'TEXTAREA' && activeElement?.tagName !== 'INPUT') {
          inputRef.current?.focus();
        }
      }, 100);
      return () => clearTimeout(timer);
    }
  }, [isChatBlocked, isSessionActive, isSending, isMediaActive, showAccessDialog, viewerFile, isEditingMessage]);

  useEffect(() => {
    const loadFingerprints = async () => {
      setIsLoadingFingerprint(true);
      try {
        const [myResponse, peerResponse] = await Promise.all([
          getFingerprint(myUserId, token),
          getFingerprint(peer.id, token),
        ]);

        setMyFingerprint(myResponse.fingerprint);
        setPeerFingerprint(peerResponse.fingerprint);

        const storedFingerprint = getVerifiedPeerFingerprint(peer.id);
        const currentFingerprint = peerResponse.fingerprint;
        const normalizedCurrent = normalizeFingerprint(currentFingerprint);

        if (!storedFingerprint) {
          setShowFingerprintModal(true);
          setIsChatBlocked(true);
          setIsFingerprintVerified(false);
        } else {
          const storedNormalized = normalizeFingerprint(storedFingerprint);
          const hasChanged = storedNormalized !== normalizedCurrent;
          const isVerified = isPeerVerified(peer.id, currentFingerprint);

          if (hasChanged && isVerified) {
            setFingerprintWarning(true);
            setIsChatBlocked(true);
            setShowFingerprintModal(true);
            setIsFingerprintVerified(false);
            showToast('Коды безопасности изменились. Пожалуйста, верифицируйте identity снова для безопасного общения.', 'error');
          } else if (!hasChanged && isVerified) {
            setIsFingerprintVerified(true);
            setIsChatBlocked(false);
            setFingerprintWarning(false);
          } else {
            setIsFingerprintVerified(false);
            setIsChatBlocked(true);
          }
        }
      } catch (err) {
        setIsFingerprintVerified(false);
        setIsChatBlocked(true);
        showToast('Не удалось загрузить fingerprint. Попробуйте перезагрузить страницу.', 'error');
      } finally {
        setIsLoadingFingerprint(false);
      }
    };

    void loadFingerprints();
  }, [token, myUserId, peer.id, showToast]);



  const handleSend = useCallback(
    async (e: React.FormEvent) => {
      e.preventDefault();
      if (!messageText.trim() || isSending || !isSessionActive || isChatBlocked) return;

      if (typingTimeoutRef.current) {
        clearTimeout(typingTimeoutRef.current);
        typingTimeoutRef.current = null;
      }
      sendTyping(false);

      setIsSending(true);
      try {
        await sendMessage(messageText.trim(), replyTo?.id);
        setMessageText('');
        if (hasShownMaxLengthToast) {
          setHasShownMaxLengthToast(false);
        }
        if (replyTo) {
          setReplyTo(null);
        }
        if (inputRef.current) {
          inputRef.current.style.height = '40px';
          if (!showAccessDialog && !viewerFile && !isEditingMessage) {
            inputRef.current.focus();
          }
        }
      } catch (err) {
        showToast('Не удалось отправить сообщение. Проверьте соединение и попробуйте снова.', 'error');
      } finally {
        setIsSending(false);
      }
    },
    [messageText, isSending, isSessionActive, isChatBlocked, sendMessage, sendTyping, showToast, replyTo],
  );

  const handleKeyDown = useCallback(
    (e: React.KeyboardEvent<HTMLTextAreaElement>) => {
      if (e.key === 'Enter' && !e.shiftKey) {
        e.preventDefault();
        handleSend(e);
      }
    },
    [handleSend],
  );

  const handleTyping = useCallback(
    (text: string) => {
      if (!isSessionActive || isChatBlocked) return;

      if (typingTimeoutRef.current) {
        clearTimeout(typingTimeoutRef.current);
      }

      if (text.trim().length > 0) {
        sendTyping(true);
        typingTimeoutRef.current = setTimeout(() => {
          sendTyping(false);
        }, 3000) as unknown as number;
      } else {
        sendTyping(false);
      }
    },
    [isSessionActive, isChatBlocked, sendTyping],
  );

  const handleMessageTextChange = useCallback(
    (e: React.ChangeEvent<HTMLTextAreaElement>) => {
      const newText = e.target.value;
      if (newText.length > MAX_MESSAGE_LENGTH) {
        if (!hasShownMaxLengthToast) {
          showToast(
            `Сообщение слишком длинное (максимум ${MAX_MESSAGE_LENGTH} символов)`,
            "warning",
          );
          setHasShownMaxLengthToast(true);
        }
        return;
      }
      setMessageText(newText);
      handleTyping(newText);
    },
    [handleTyping, hasShownMaxLengthToast, showToast],
  );

  const handleInputBlur = useCallback(() => {
    if (typingTimeoutRef.current) {
      clearTimeout(typingTimeoutRef.current);
      typingTimeoutRef.current = null;
    }
    sendTyping(false);

    if (!isChatBlocked && isSessionActive && !isSending && inputRef.current && !isMediaActive && !showAccessDialog && !viewerFile && !isEditingMessage) {
      setTimeout(() => {
        const isEditing = document.querySelector('textarea[data-edit-input]') !== null;
        if (inputRef.current && !isEditing && document.activeElement !== inputRef.current && document.activeElement?.tagName !== 'TEXTAREA' && document.activeElement?.tagName !== 'INPUT') {
          inputRef.current.focus();
        }
      }, 100);
    }
  }, [isChatBlocked, isSessionActive, isSending, isMediaActive, sendTyping, showAccessDialog, viewerFile, isEditingMessage]);

  const handleFileSelect = useCallback(
    async (e: React.ChangeEvent<HTMLInputElement>) => {
      const file = e.target.files?.[0];
      if (!file || isSendingFile || !isSessionActive || isChatBlocked) return;

      if (file.size > MAX_FILE_SIZE) {
        showToast('Файл слишком большой. Максимальный размер: 50MB. Выберите файл меньшего размера.', 'error');
        return;
      }

      setPendingFile(file);
      if (fileInputRef.current) {
        fileInputRef.current.value = '';
        fileInputRef.current.blur();
      }
      setShowAccessDialog(true);
    },
    [isSendingFile, isSessionActive, isChatBlocked, sendFile, showToast],
  );

  const handlePaste = useCallback(
    async (e: React.ClipboardEvent<HTMLTextAreaElement>) => {
      if (isSendingFile || !isSessionActive || isChatBlocked) return;

      const items = e.clipboardData.items;
      for (let i = 0; i < items.length; i++) {
        const item = items[i];
        if (item.type.startsWith('image/')) {
          e.preventDefault();
          const file = item.getAsFile();
          if (!file) continue;

          if (file.size > MAX_FILE_SIZE) {
            showToast('Изображение слишком большое. Максимальный размер: 50MB. Выберите изображение меньшего размера.', 'error');
            continue;
          }

          if (inputRef.current) {
            inputRef.current.blur();
          }
          setPendingFile(file);
          setShowAccessDialog(true);
          break;
        }
      }
    },
    [isSendingFile, isSessionActive, isChatBlocked, showToast],
  );


  const handleFileButtonClick = useCallback(() => {
    if (!isSessionActive || isChatBlocked) return;
    fileInputRef.current?.click();
  }, [isSessionActive, isChatBlocked]);

  const stateMessage = useMemo(() => {
    switch (state) {
      case 'establishing':
        return 'Установка защищённой сессии...';
      case 'peer_offline':
        return 'Собеседник не в сети';
      case 'peer_disconnected':
        return 'Собеседник отключился';
      case 'error':
        return error || 'Ошибка соединения';
      default:
        return null;
    }
  }, [state, error]);

  const isLoading = state === 'establishing';

  const handleMediaActiveChange = useCallback((active: boolean) => {
    setActiveMediaCount((current) => {
      const next = current + (active ? 1 : -1);
      return next < 0 ? 0 : next;
    });
  }, []);

  useEffect(() => {
    const handleEscape = (e: KeyboardEvent) => {
      if (e.key === 'Escape' && !showFingerprintModal) {
        onClose();
      }
    };

    window.addEventListener('keydown', handleEscape);
    return () => {
      window.removeEventListener('keydown', handleEscape);
    };
  }, [onClose, showFingerprintModal]);

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/80 backdrop-blur-sm smooth-transition" onClick={(e) => e.target === e.currentTarget && onClose()}>
      <div className="w-full max-w-2xl h-[80vh] flex flex-col bg-black border border-emerald-700 rounded-xl overflow-hidden modal-enter glow-emerald" style={{ willChange: 'transform, opacity' }}>
        <div className="flex items-center justify-between px-4 py-3 border-b border-emerald-700/60 bg-black/80">
          <div className="flex items-center gap-3">
            <button
              type="button"
              onClick={onClose}
              className="text-emerald-400 hover:text-emerald-200 smooth-transition button-press rounded-md p-1 hover:bg-emerald-900/40"
              aria-label="Закрыть чат"
            >
              <svg
                className="w-5 h-5"
                fill="none"
                stroke="currentColor"
                viewBox="0 0 24 24"
              >
                <path
                  strokeLinecap="round"
                  strokeLinejoin="round"
                  strokeWidth={2}
                  d="M15 19l-7-7 7-7"
                />
              </svg>
            </button>
            <div>
              <h2 className="text-sm font-semibold text-emerald-300 tracking-tight">{peer.username}</h2>
              <div className="flex items-center gap-2 mt-0.5">
                {isLoadingFingerprint ? (
                  <span className="text-[10px] text-emerald-500/60">Проверка безопасности...</span>
                ) : isSessionActive ? (
                  <>
                    <span className="inline-flex h-1.5 w-1.5 rounded-full bg-emerald-400 animate-pulse" />
                    <span className="text-[10px] text-emerald-500/80">Защищённая сессия</span>
                  </>
                ) : (
                  <span className="text-[10px] text-emerald-500/60">
                    {stateMessage || 'Подключение...'}
                  </span>
                )}
              </div>
            </div>
          </div>
          {isSessionActive && (
            <button
              type="button"
              onClick={() => setShowFingerprintModal(true)}
              className={`px-3 py-1.5 text-xs font-medium rounded-md transition-colors ${
                fingerprintWarning
                  ? 'bg-yellow-500/20 hover:bg-yellow-500/30 text-yellow-400 border border-yellow-700/40'
                  : isPeerVerified(peer.id, peerFingerprint || '')
                    ? 'bg-emerald-500/20 hover:bg-emerald-500/30 text-emerald-400 border border-emerald-700/40'
                    : 'bg-emerald-900/40 hover:bg-emerald-900/60 text-emerald-300 border border-emerald-700/60'
              }`}
            >
              {fingerprintWarning ? (
                <span className="flex items-center gap-1.5">
                  <svg className="w-3.5 h-3.5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                    <path
                      strokeLinecap="round"
                      strokeLinejoin="round"
                      strokeWidth={2}
                      d="M12 9v2m0 4h.01m-6.938 4h13.856c1.54 0 2.502-1.667 1.732-3L13.732 4c-.77-1.333-2.694-1.333-3.464 0L3.34 16c-.77 1.333.192 3 1.732 3z"
                    />
                  </svg>
                  Проверить Identity
                </span>
              ) : isPeerVerified(peer.id, peerFingerprint || '') ? (
                <span className="flex items-center gap-1.5">
                  <svg className="w-3.5 h-3.5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                    <path
                      strokeLinecap="round"
                      strokeLinejoin="round"
                      strokeWidth={2}
                      d="M9 12l2 2 4-4m6 2a9 9 0 11-18 0 9 9 0 0118 0z"
                    />
                  </svg>
                  Подтверждён
                </span>
              ) : (
                'Проверить Identity'
              )}
            </button>
          )}
        </div>

        <div className="flex-1 overflow-y-auto px-4 py-4 space-y-3 scrollbar-custom relative chat-scroll-area">
          {isLoadingFingerprint && (
            <div className="flex items-center justify-center py-8">
              <div className="flex flex-col items-center gap-3">
                <div className="w-8 h-8 border-2 border-emerald-400 border-t-transparent rounded-full animate-spin" />
                <p className="text-xs text-emerald-500/80">Проверка безопасности...</p>
              </div>
            </div>
          )}

          {!isLoadingFingerprint && isLoading && (
            <div className="flex items-center justify-center py-8">
              <div className="flex flex-col items-center gap-3">
                <div className="w-8 h-8 border-2 border-emerald-400 border-t-transparent rounded-full animate-spin" />
                <p className="text-xs text-emerald-500/80">{stateMessage}</p>
              </div>
            </div>
          )}

          {!isLoading && messages.length === 0 && isSessionActive && (
            <div className="flex items-center justify-center py-8">
              <p className="text-xs text-emerald-500/60">
                Начните переписку. Все сообщения зашифрованы.
              </p>
            </div>
          )}

          {messages.map((message) => (
            <MessageBubble
              key={message.id}
              message={message}
              myUserId={myUserId}
              peerUsername={peer.username}
              onReaction={sendReaction}
              onDelete={deleteMessage}
              onEdit={editMessage}
              onMarkAsRead={markMessageAsRead}
              onMediaActiveChange={handleMediaActiveChange}
              onReply={setReplyTo}
              onEditingChange={setIsEditingMessage}
              onViewFile={(filename, mimeType, blob, isProtected) => {
                if (inputRef.current) {
                  inputRef.current.blur();
                }
                setViewerFile({ filename, mimeType, blob, isProtected });
              }}
            />
          ))}

          <TypingIndicator isVisible={isPeerTyping && isSessionActive && !isChatBlocked} />

          {isChatBlocked && (
            <div className="flex items-center justify-center py-4">
              <div className="bg-yellow-900/20 border border-yellow-700/40 rounded-lg px-4 py-3 max-w-md">
                <div className="flex items-start gap-2">
                  <svg
                    className="w-5 h-5 text-yellow-400 mt-0.5 flex-shrink-0"
                    fill="none"
                    stroke="currentColor"
                    viewBox="0 0 24 24"
                  >
                    <path
                      strokeLinecap="round"
                      strokeLinejoin="round"
                      strokeWidth={2}
                      d="M12 9v2m0 4h.01m-6.938 4h13.856c1.54 0 2.502-1.667 1.732-3L13.732 4c-.77-1.333-2.694-1.333-3.464 0L3.34 16c-.77 1.333.192 3 1.732 3z"
                    />
                  </svg>
                  <div>
                    <p className="text-sm font-medium text-yellow-400">
                      Чат заблокирован: Security codes изменились!
                    </p>
                    <p className="text-xs text-yellow-500/80 mt-1">
                      Верифицируйте identity собеседника, чтобы продолжить общение.
                    </p>
                  </div>
                </div>
              </div>
            </div>
          )}

          {error && state === 'error' && (
            <div className="flex items-center justify-center py-4">
              <p className="text-xs text-red-400 bg-red-900/20 border border-red-700/40 rounded px-3 py-2">
                {error}
              </p>
            </div>
          )}

          <div ref={messagesEndRef} />
        </div>

        <form
          onSubmit={handleSend}
          className="border-t border-emerald-700/60 bg-black/80 px-4 py-3"
        >
          {replyTo && (
            <div className="mb-2 rounded-lg border border-emerald-700/60 bg-black/70 px-3 py-2 flex items-start justify-between gap-2">
              <div className="flex-1 min-w-0">
                <p className="text-[11px] text-emerald-400/80">Ответ на сообщение</p>
                <p className="text-xs text-emerald-50 truncate">
                  {replyTo.text
                    ? replyTo.text
                    : replyTo.voice
                      ? 'Голосовое сообщение'
                      : replyTo.file
                        ? 'Файл'
                        : 'Сообщение'}
                </p>
              </div>
              <button
                type="button"
                onClick={() => setReplyTo(null)}
                className="flex-shrink-0 w-5 h-5 rounded-full bg-emerald-800/60 hover:bg-emerald-700 text-emerald-50 flex items-center justify-center text-[10px] transition-colors mt-1"
                aria-label="Отменить ответ"
              >
                ×
              </button>
            </div>
          )}
          <input
            ref={fileInputRef}
            type="file"
            accept="image/*,.pdf,.doc,.docx,.xls,.xlsx,.ppt,.pptx,.txt,.rtf,.odt,.ods,.odp"
            onChange={handleFileSelect}
            className="hidden"
            disabled={!isSessionActive || isChatBlocked}
          />
          <div className="flex items-center gap-2">
            <VoiceRecorder
              onRecorded={async (file, duration) => {
                if (!isSessionActive || isChatBlocked) return;
                try {
                  await sendVoice(file, duration);
                  showToast('Голосовое сообщение отправлено', 'success');
                } catch {
                  showToast('Не удалось отправить голосовое сообщение. Проверьте микрофон и попробуйте снова.', 'error');
                }
              }}
              onError={(error) => showToast(error, 'error')}
              disabled={!isSessionActive || isChatBlocked}
            />
            <div className="flex-1 relative flex items-center">
              <textarea
                ref={inputRef}
                value={messageText}
                onChange={handleMessageTextChange}
                onKeyDown={handleKeyDown}
                onPaste={handlePaste}
                onBlur={handleInputBlur}
                disabled={!isSessionActive || isSending || isChatBlocked}
                placeholder={
                  isChatBlocked
                    ? 'Чат заблокирован: верифицируйте identity'
                    : isSessionActive
                      ? 'Введите сообщение...'
                      : 'Ожидание установки сессии...'
                }
                rows={1}
                className="w-full resize-none rounded-md bg-black border border-emerald-700 px-3 pr-10 py-2.5 text-sm text-emerald-50 placeholder-emerald-500/50 outline-none focus:ring-2 focus:ring-emerald-500 disabled:opacity-50 disabled:cursor-not-allowed transition-all scrollbar-custom"
                style={{
                  maxHeight: '120px',
                  minHeight: '40px',
                  height: '40px',
                }}
                onInput={(e) => {
                  const target = e.currentTarget;
                  target.style.height = 'auto';
                  const newHeight = Math.min(target.scrollHeight, 120);
                  target.style.height = `${newHeight}px`;
                }}
              />
              <button
                type="button"
                onClick={handleFileButtonClick}
                disabled={!isSessionActive || isSendingFile || isChatBlocked}
                className="absolute right-2 rounded-md bg-emerald-900/40 hover:bg-emerald-900/60 disabled:bg-emerald-900/20 disabled:cursor-not-allowed text-emerald-300 p-1.5 smooth-transition button-press flex items-center justify-center h-7 w-7"
                style={{ top: '50%', transform: 'translateY(-50%)' }}
                title="Прикрепить файл"
              >
                {isSendingFile ? (
                  <div className="w-3.5 h-3.5 border-2 border-emerald-300 border-t-transparent rounded-full animate-spin" />
                ) : (
                  <svg
                    className="w-4 h-4"
                    fill="none"
                    stroke="currentColor"
                    viewBox="0 0 24 24"
                  >
                    <path
                      strokeLinecap="round"
                      strokeLinejoin="round"
                      strokeWidth={2}
                      d="M15.172 7l-6.586 6.586a2 2 0 102.828 2.828l6.414-6.586a4 4 0 00-5.656-5.656l-6.415 6.585a6 6 0 108.486 8.486L20.5 13"
                    />
                  </svg>
                )}
              </button>
            </div>
            <button
              type="submit"
              disabled={!messageText.trim() || !isSessionActive || isSending}
              className="rounded-md bg-emerald-500 hover:bg-emerald-400 disabled:bg-emerald-700 disabled:cursor-not-allowed text-sm font-medium px-4 h-10 text-black smooth-transition button-press glow-emerald-hover flex items-center justify-center min-w-[80px]"
            >
              {isSending ? (
                <div className="w-4 h-4 border-2 border-black border-t-transparent rounded-full animate-spin" />
              ) : (
                'Отправить'
              )}
            </button>
          </div>
        </form>
      </div>

      {showFingerprintModal && (
        <FingerprintVerificationModal
          token={token}
          peerId={peer.id}
          peerUsername={peer.username}
          myFingerprint={myFingerprint}
          onClose={() => {
            setShowFingerprintModal(false);
            const isVerified = isPeerVerified(peer.id, peerFingerprint || '');
            if (isVerified) {
              setFingerprintWarning(false);
              setIsChatBlocked(false);
              setIsFingerprintVerified(true);
            } else {
              setIsChatBlocked(true);
              setIsFingerprintVerified(false);
            }
          }}
          onVerified={() => {
            setFingerprintWarning(false);
            setIsChatBlocked(false);
            setIsFingerprintVerified(true);
            showToast('Identity подтверждён. Безопасное общение установлено.', 'success');
          }}
        />
      )}

      {showAccessDialog && pendingFile && (
        <FileAccessDialog
          filename={pendingFile.name}
          onSelect={async (mode: FileAccessMode) => {
            setShowAccessDialog(false);
            setIsSendingFile(true);
            try {
              await sendFile(pendingFile, mode);
              showToast('Файл отправлен', 'success');
            } catch (err) {
              showToast('Не удалось отправить файл. Проверьте соединение и попробуйте снова.', 'error');
            } finally {
              setIsSendingFile(false);
              setPendingFile(null);
            }
          }}
          onCancel={() => {
            setShowAccessDialog(false);
            setPendingFile(null);
          }}
        />
      )}

      {viewerFile && (
        <FileViewerModal
          filename={viewerFile.filename}
          mimeType={viewerFile.mimeType}
          blob={viewerFile.blob}
          onClose={() => setViewerFile(null)}
          protected={viewerFile.isProtected}
        />
      )}
    </div>
  );
}
