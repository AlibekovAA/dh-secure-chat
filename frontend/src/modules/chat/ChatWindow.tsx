import { useCallback, useEffect, useMemo, useRef, useState } from 'react';
import { useChatSession } from './useChatSession';
import type { UserSummary } from './api';
import { FingerprintVerificationModal } from './FingerprintVerificationModal';
import { FileMessage } from './FileMessage';
import { VoiceMessage } from './VoiceMessage';
import { VoiceRecorder } from './VoiceRecorder';
import { getFingerprint } from './api';
import {
  getVerifiedPeerFingerprint,
  isPeerVerified,
} from '../../shared/crypto/fingerprint';
import { useToast } from '../../shared/ui/ToastProvider';
import { MAX_FILE_SIZE } from './constants';

type Props = {
  token: string;
  peer: UserSummary;
  myUserId: string;
  onClose(): void;
};

export function ChatWindow({ token, peer, myUserId, onClose }: Props) {
  const [messageText, setMessageText] = useState('');
  const [isSending, setIsSending] = useState(false);
  const [isSendingFile, setIsSendingFile] = useState(false);
  const fileInputRef = useRef<HTMLInputElement>(null);
  const [showFingerprintModal, setShowFingerprintModal] = useState(false);
  const [myFingerprint, setMyFingerprint] = useState<string | null>(null);
  const [peerFingerprint, setPeerFingerprint] = useState<string | null>(null);
  const [fingerprintWarning, setFingerprintWarning] = useState(false);
  const [isChatBlocked, setIsChatBlocked] = useState(false);
  const [imagePreview, setImagePreview] = useState<{ file: File; url: string } | null>(null);
  const messagesEndRef = useRef<HTMLDivElement>(null);
  const inputRef = useRef<HTMLTextAreaElement>(null);
  const { showToast } = useToast();

  const { state, messages, error, sendMessage, sendFile, sendVoice, isSessionActive } =
    useChatSession({
      token,
      peerId: peer.id,
      enabled: true,
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
    if (inputRef.current && !isChatBlocked && isSessionActive && !isSending) {
      const timer = setTimeout(() => {
        inputRef.current?.focus();
      }, 100);
      return () => clearTimeout(timer);
    }
  }, [isChatBlocked, isSessionActive, isSending]);

  useEffect(() => {
    const loadFingerprints = async () => {
      try {
        const [myResponse, peerResponse] = await Promise.all([
          getFingerprint(myUserId, token),
          getFingerprint(peer.id, token),
        ]);

        setMyFingerprint(myResponse.fingerprint);
        setPeerFingerprint(peerResponse.fingerprint);

        const storedFingerprint = getVerifiedPeerFingerprint(peer.id);
        const currentFingerprint = peerResponse.fingerprint;

        if (!storedFingerprint) {
          setShowFingerprintModal(true);
          setIsChatBlocked(true);
        } else {
          const hasChanged = storedFingerprint !== currentFingerprint;
          const isVerified = isPeerVerified(peer.id, currentFingerprint);

          if (hasChanged && isVerified) {
            setFingerprintWarning(true);
            setIsChatBlocked(true);
            setShowFingerprintModal(true);
            showToast('Security codes изменились! Верифицируйте identity снова.', 'error');
          }
        }
      } catch (err) {
        console.warn('Failed to load fingerprints:', err);
      }
    };

    if (isSessionActive) {
      void loadFingerprints();
    }
  }, [token, myUserId, peer.id, isSessionActive, showToast]);

  useEffect(() => {
    return () => {
      if (imagePreview) {
        URL.revokeObjectURL(imagePreview.url);
      }
    };
  }, [imagePreview]);

  const handleRemovePreview = useCallback(() => {
    if (imagePreview) {
      URL.revokeObjectURL(imagePreview.url);
      setImagePreview(null);
    }
  }, [imagePreview]);


  const handleSend = useCallback(
    async (e: React.FormEvent) => {
      e.preventDefault();
      if ((!messageText.trim() && !imagePreview) || isSending || !isSessionActive || isChatBlocked) return;

      setIsSending(true);
      try {
        if (imagePreview) {
          await sendFile(imagePreview.file);
          handleRemovePreview();
        } else {
          await sendMessage(messageText.trim());
        }
        setMessageText('');
        if (inputRef.current) {
          inputRef.current.style.height = '40px';
          inputRef.current.focus();
        }
      } catch (err) {
        showToast('Ошибка отправки сообщения', 'error');
      } finally {
        setIsSending(false);
      }
    },
    [messageText, imagePreview, isSending, isSessionActive, isChatBlocked, sendMessage, sendFile, handleRemovePreview, showToast],
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

  const handleInputBlur = useCallback(() => {
    if (!isChatBlocked && isSessionActive && !isSending && inputRef.current) {
      setTimeout(() => {
        if (inputRef.current && document.activeElement !== inputRef.current) {
          inputRef.current.focus();
        }
      }, 100);
    }
  }, [isChatBlocked, isSessionActive, isSending]);

  const handleFileSelect = useCallback(
    async (e: React.ChangeEvent<HTMLInputElement>) => {
      const file = e.target.files?.[0];
      if (!file || isSendingFile || !isSessionActive || isChatBlocked) return;

      if (file.size > MAX_FILE_SIZE) {
        showToast('Файл слишком большой (максимум 50MB)', 'error');
        return;
      }

      if (file.type.startsWith('image/')) {
        if (imagePreview) {
          URL.revokeObjectURL(imagePreview.url);
        }
        const url = URL.createObjectURL(file);
        setImagePreview({ file, url });
        if (fileInputRef.current) {
          fileInputRef.current.value = '';
        }
        return;
      }

      setIsSendingFile(true);
      try {
        await sendFile(file);
        showToast('Файл отправлен', 'success');
      } catch (err) {
        showToast('Ошибка отправки файла', 'error');
      } finally {
        setIsSendingFile(false);
        if (fileInputRef.current) {
          fileInputRef.current.value = '';
        }
      }
    },
    [isSendingFile, isSessionActive, isChatBlocked, sendFile, showToast, imagePreview],
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
            showToast('Изображение слишком большое (максимум 50MB)', 'error');
            continue;
          }

          if (imagePreview) {
            URL.revokeObjectURL(imagePreview.url);
          }

          const url = URL.createObjectURL(file);
          setImagePreview({ file, url });
          break;
        }
      }
    },
    [isSendingFile, isSessionActive, isChatBlocked, showToast, imagePreview],
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

  useEffect(() => {
    return () => {
      if (imagePreview) {
        URL.revokeObjectURL(imagePreview.url);
      }
    };
  }, [imagePreview]);

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/80 backdrop-blur-sm" onClick={(e) => e.target === e.currentTarget && onClose()}>
      <div className="w-full max-w-2xl h-[80vh] flex flex-col bg-black border border-emerald-700 rounded-xl overflow-hidden animate-[fadeIn_0.3s_ease-out,slideUp_0.3s_ease-out]">
        <div className="flex items-center justify-between px-4 py-3 border-b border-emerald-700/60 bg-black/80">
          <div className="flex items-center gap-3">
            <button
              type="button"
              onClick={onClose}
              className="text-emerald-400 hover:text-emerald-200 transition-colors"
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
              <h2 className="text-sm font-semibold text-emerald-300">{peer.username}</h2>
              <div className="flex items-center gap-2 mt-0.5">
                {isSessionActive ? (
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

        <div className="flex-1 overflow-y-auto px-4 py-4 space-y-3 scrollbar-custom">
          {isLoading && (
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
            <div
              key={message.id}
              className={`flex ${message.isOwn ? 'justify-end' : 'justify-start'} animate-[fadeIn_0.2s_ease-out]`}
            >
              <div
                className={`max-w-[75%] rounded-lg px-3 py-2 ${
                  message.isOwn
                    ? 'bg-emerald-500/20 border border-emerald-500/40 text-emerald-50'
                    : 'bg-emerald-900/20 border border-emerald-700/40 text-emerald-100'
                }`}
              >
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
                  />
                )}
                {message.file && !message.voice && (
                  <FileMessage
                    filename={message.file.filename}
                    mimeType={message.file.mimeType}
                    size={message.file.size}
                    blob={message.file.blob}
                    isOwn={message.isOwn}
                  />
                )}
                <p className="text-[10px] text-emerald-500/60 mt-1">
                  {new Date(message.timestamp).toLocaleTimeString('ru-RU', {
                    hour: '2-digit',
                    minute: '2-digit',
                  })}
                </p>
              </div>
            </div>
          ))}

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

        {imagePreview && (
          <div className="border-t border-emerald-700/60 bg-black/80 px-4 py-3">
            <div className="flex items-center gap-3 p-3 rounded-lg border border-emerald-700/40 bg-emerald-900/20">
              <div className="relative flex-shrink-0">
                <img
                  src={imagePreview.url}
                  alt="Preview"
                  className="w-16 h-16 object-cover rounded border border-emerald-700/40"
                />
                <button
                  type="button"
                  onClick={handleRemovePreview}
                  className="absolute -top-1 -right-1 w-5 h-5 rounded-full bg-red-500 hover:bg-red-600 text-white flex items-center justify-center text-xs transition-colors"
                  aria-label="Удалить предпросмотр"
                >
                  ×
                </button>
              </div>
              <div className="flex-1 min-w-0">
                <p className="text-sm text-emerald-100 truncate">{imagePreview.file.name}</p>
                <p className="text-xs text-emerald-500/80 mt-0.5">
                  {(imagePreview.file.size / 1024 / 1024).toFixed(2)} MB
                </p>
              </div>
            </div>
          </div>
        )}

        <form
          onSubmit={handleSend}
          className="border-t border-emerald-700/60 bg-black/80 px-4 py-3"
        >
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
                } catch (err) {
                  showToast('Ошибка отправки голосового сообщения', 'error');
                }
              }}
              onError={(error) => showToast(error, 'error')}
              disabled={!isSessionActive || isChatBlocked}
            />
            <div className="flex-1 relative flex items-center">
              <textarea
                ref={inputRef}
                value={messageText}
                onChange={(e) => setMessageText(e.target.value)}
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
                className="absolute right-2 rounded-md bg-emerald-900/40 hover:bg-emerald-900/60 disabled:bg-emerald-900/20 disabled:cursor-not-allowed text-emerald-300 p-1.5 transition-colors flex items-center justify-center h-7 w-7"
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
              disabled={(!messageText.trim() && !imagePreview) || !isSessionActive || isSending}
              className="rounded-md bg-emerald-500 hover:bg-emerald-400 disabled:bg-emerald-700 disabled:cursor-not-allowed text-sm font-medium px-4 h-10 text-black transition-colors flex items-center justify-center min-w-[80px]"
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
            } else {
              setIsChatBlocked(true);
            }
          }}
          onVerified={() => {
            setFingerprintWarning(false);
            setIsChatBlocked(false);
            showToast('Identity подтверждён. Безопасное общение установлено.', 'success');
          }}
        />
      )}
    </div>
  );
}
