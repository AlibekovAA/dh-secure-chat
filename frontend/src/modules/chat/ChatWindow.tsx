import { useCallback, useEffect, useMemo, useRef, useState } from 'react';
import {
  useChatSession,
  type ChatMessage,
} from '@/modules/chat/useChatSession';
import type { UserSummary } from '@/modules/chat/api';
import { FingerprintVerificationModal } from '@/modules/chat/FingerprintVerificationModal';
import { VoiceRecorder } from '@/modules/chat/VoiceRecorder';
import { VideoRecorder } from '@/modules/chat/VideoRecorder';
import { MessageList } from '@/modules/chat/MessageList';
import {
  FileAccessDialog,
  type FileAccessMode,
} from '@/modules/chat/FileAccessDialog';
import { FileViewerModal } from '@/modules/chat/FileViewerModal';
import { getFingerprint } from '@/modules/chat/api';
import {
  getVerifiedPeerFingerprint,
  isPeerVerified,
  normalizeFingerprint,
} from '@/shared/crypto/fingerprint';
import { getFriendlyErrorMessage } from '@/shared/api/error-handler';
import { Spinner } from '@/shared/ui/Spinner';
import { useToast } from '@/shared/ui/useToast';
import {
  MAX_MESSAGE_LENGTH,
  TYPING_INDICATOR_TIMEOUT_MS,
  INPUT_MIN_HEIGHT_PX,
  MODAL_MAX_HEIGHT_VH,
} from '@/shared/constants';
import {
  validateFileSize,
  validateImagePaste,
  getFileValidationError,
} from '@/shared/utils/files';
import { attachTokenToClient } from '@/shared/api/session';
import { MESSAGES } from '@/shared/messages';

type Props = {
  token: string;
  peer: UserSummary;
  myUserId: string;
  onClose(): void;
  onTokenExpired?: () => Promise<string | null>;
};

export function ChatWindow({
  token,
  peer,
  myUserId,
  onClose,
  onTokenExpired,
}: Props) {
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
  const inputRef = useRef<HTMLTextAreaElement>(null);
  const typingTimeoutRef = useRef<number | null>(null);
  const { showToast } = useToast();
  const [activeMediaCount, setActiveMediaCount] = useState(0);
  const isMediaActive = activeMediaCount > 0;
  const [isVoiceRecording, setIsVoiceRecording] = useState(false);
  const [hasShownMaxLengthToast, setHasShownMaxLengthToast] = useState(false);
  const [replyTo, setReplyTo] = useState<ChatMessage | null>(null);
  const [viewerFile, setViewerFile] = useState<{
    filename: string;
    mimeType: string;
    blob: Blob;
    isProtected: boolean;
  } | null>(null);
  const [isEditingMessage, setIsEditingMessage] = useState(false);
  const scrollContainerRef = useRef<HTMLDivElement>(null);

  useEffect(() => {
    if (token) {
      attachTokenToClient(token);
    }
  }, [token]);

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
    peerActivity,
    isSessionActive,
  } = useChatSession({
    token,
    peerId: peer.id,
    enabled: isFingerprintVerified,
    onTokenExpired,
  });

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
        const isEditing =
          document.querySelector('textarea[data-edit-input]') !== null;
        if (
          !isEditing &&
          activeElement?.tagName !== 'TEXTAREA' &&
          activeElement?.tagName !== 'INPUT'
        ) {
          inputRef.current?.focus();
        }
      }, 100);
      return () => clearTimeout(timer);
    }
  }, [
    isChatBlocked,
    isSessionActive,
    isSending,
    isMediaActive,
    showAccessDialog,
    viewerFile,
    isEditingMessage,
  ]);

  useEffect(() => {
    const loadFingerprints = async () => {
      setIsLoadingFingerprint(true);
      try {
        const [myResponse, peerResponse] = await Promise.all([
          getFingerprint(myUserId),
          getFingerprint(peer.id),
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
            showToast(
              MESSAGES.chat.window.warnings.securityCodesChanged,
              'error'
            );
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
        showToast(MESSAGES.chat.window.errors.failedToLoadFingerprint, 'error');
      } finally {
        setIsLoadingFingerprint(false);
      }
    };

    void loadFingerprints();
  }, [token, myUserId, peer.id, showToast]);

  const handleSend = useCallback(
    async (e: React.FormEvent) => {
      e.preventDefault();
      if (!messageText.trim() || isSending || !isSessionActive || isChatBlocked)
        return;

      if (typingTimeoutRef.current) {
        clearTimeout(typingTimeoutRef.current);
        typingTimeoutRef.current = null;
      }
      sendTyping(null);

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
          inputRef.current.style.height = `${INPUT_MIN_HEIGHT_PX}px`;
          if (!showAccessDialog && !viewerFile && !isEditingMessage) {
            inputRef.current.focus();
          }
        }
      } catch (err) {
        const errorMessage = getFriendlyErrorMessage(
          err,
          MESSAGES.chat.window.errors.failedToSendMessage
        );
        showToast(errorMessage, 'error');
      } finally {
        setIsSending(false);
      }
    },
    [
      messageText,
      isSending,
      isSessionActive,
      isChatBlocked,
      sendMessage,
      sendTyping,
      showToast,
      replyTo,
      hasShownMaxLengthToast,
      showAccessDialog,
      viewerFile,
      isEditingMessage,
    ]
  );

  const handleKeyDown = useCallback(
    (e: React.KeyboardEvent<HTMLTextAreaElement>) => {
      if (e.key === 'Enter' && !e.shiftKey) {
        e.preventDefault();
        handleSend(e);
      }
    },
    [handleSend]
  );

  const handleTyping = useCallback(
    (text: string) => {
      if (!isSessionActive || isChatBlocked) return;

      if (typingTimeoutRef.current) {
        clearTimeout(typingTimeoutRef.current);
      }

      if (text.trim().length > 0) {
        sendTyping('typing');
        typingTimeoutRef.current = setTimeout(() => {
          sendTyping(null);
        }, TYPING_INDICATOR_TIMEOUT_MS) as unknown as number;
      } else {
        sendTyping(null);
      }
    },
    [isSessionActive, isChatBlocked, sendTyping]
  );

  const handleMessageTextChange = useCallback(
    (e: React.ChangeEvent<HTMLTextAreaElement>) => {
      const newText = e.target.value;
      if (newText.length > MAX_MESSAGE_LENGTH) {
        if (!hasShownMaxLengthToast) {
          showToast(
            MESSAGES.chat.window.errors.messageTooLong(MAX_MESSAGE_LENGTH),
            'warning'
          );
          setHasShownMaxLengthToast(true);
        }
        return;
      }
      setMessageText(newText);
      handleTyping(newText);
    },
    [handleTyping, hasShownMaxLengthToast, showToast]
  );

  const handleInputBlur = useCallback(() => {
    if (typingTimeoutRef.current) {
      clearTimeout(typingTimeoutRef.current);
      typingTimeoutRef.current = null;
    }
    sendTyping(null);

    if (
      !isChatBlocked &&
      isSessionActive &&
      !isSending &&
      inputRef.current &&
      !isMediaActive &&
      !showAccessDialog &&
      !viewerFile &&
      !isEditingMessage
    ) {
      setTimeout(() => {
        const isEditing =
          document.querySelector('textarea[data-edit-input]') !== null;
        if (
          inputRef.current &&
          !isEditing &&
          document.activeElement !== inputRef.current &&
          document.activeElement?.tagName !== 'TEXTAREA' &&
          document.activeElement?.tagName !== 'INPUT'
        ) {
          inputRef.current.focus();
        }
      }, 100);
    }
  }, [
    isChatBlocked,
    isSessionActive,
    isSending,
    isMediaActive,
    sendTyping,
    showAccessDialog,
    viewerFile,
    isEditingMessage,
  ]);

  const handleFileSelect = useCallback(
    async (e: React.ChangeEvent<HTMLInputElement>) => {
      const file = e.target.files?.[0];
      if (!file || isSendingFile || !isSessionActive || isChatBlocked) return;

      const validationError = validateFileSize(file, { fileType: 'file' });
      if (validationError) {
        showToast(getFileValidationError(validationError)!, 'error');
        return;
      }

      setPendingFile(file);
      if (fileInputRef.current) {
        fileInputRef.current.value = '';
        fileInputRef.current.blur();
      }
      setShowAccessDialog(true);
    },
    [isSendingFile, isSessionActive, isChatBlocked, showToast]
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

          const validationError = validateImagePaste(file);
          if (validationError) {
            showToast(getFileValidationError(validationError)!, 'error');
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
    [isSendingFile, isSessionActive, isChatBlocked, showToast]
  );

  const handleFileButtonClick = useCallback(() => {
    if (!isSessionActive || isChatBlocked) return;
    fileInputRef.current?.click();
  }, [isSessionActive, isChatBlocked]);

  const stateMessage = useMemo(() => {
    switch (state) {
      case 'establishing':
        return MESSAGES.chat.window.status.establishingSecureSession;
      case 'peer_offline':
        return MESSAGES.chat.window.status.peerOffline;
      case 'peer_disconnected':
        return MESSAGES.chat.window.status.peerDisconnected;
      case 'error':
        return error || MESSAGES.chat.window.status.connectionError;
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
    <div
      className="fixed inset-0 z-50 flex items-center justify-center bg-black/80 backdrop-blur-sm animate-[backdropEnter_0.2s_ease-out]"
      onClick={(e) => e.target === e.currentTarget && onClose()}
    >
      <div
        className="w-full max-w-2xl flex flex-col bg-black border border-emerald-700 rounded-xl overflow-hidden animate-[modalEnter_0.3s_cubic-bezier(0.4,0,0.2,1)] glow-emerald"
        style={{
          willChange: 'transform, opacity',
          height: `${MODAL_MAX_HEIGHT_VH}vh`,
        }}
      >
        <div className="flex items-center justify-between px-4 py-3 border-b border-emerald-700/60 bg-black/80">
          <div className="flex items-center gap-3">
            <button
              type="button"
              onClick={onClose}
              className="text-emerald-400 hover:text-emerald-200 smooth-transition button-press rounded-md p-1 hover:bg-emerald-900/40"
              aria-label={MESSAGES.chat.window.aria.closeChat}
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
              <h2 className="text-sm font-semibold text-emerald-300 tracking-tight">
                {peer.username}
              </h2>
              <div className="flex items-center gap-2 mt-0.5">
                {isLoadingFingerprint ? (
                  <span className="text-[10px] text-emerald-500/60">
                    {MESSAGES.chat.window.labels.securityCheck}
                  </span>
                ) : isSessionActive ? (
                  <>
                    <span className="inline-flex h-1.5 w-1.5 rounded-full bg-emerald-400 animate-pulse" />
                    <span className="text-[10px] text-emerald-500/80">
                      {MESSAGES.chat.window.labels.secureSession}
                    </span>
                  </>
                ) : (
                  <span className="text-[10px] text-emerald-500/60">
                    {stateMessage || MESSAGES.chat.window.status.connecting}
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
                  <svg
                    className="w-3.5 h-3.5"
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
                  {MESSAGES.chat.window.actions.verifyIdentity}
                </span>
              ) : isPeerVerified(peer.id, peerFingerprint || '') ? (
                <span className="flex items-center gap-1.5">
                  <svg
                    className="w-3.5 h-3.5"
                    fill="none"
                    stroke="currentColor"
                    viewBox="0 0 24 24"
                  >
                    <path
                      strokeLinecap="round"
                      strokeLinejoin="round"
                      strokeWidth={2}
                      d="M9 12l2 2 4-4m6 2a9 9 0 11-18 0 9 9 0 0118 0z"
                    />
                  </svg>
                  {MESSAGES.chat.window.actions.verified}
                </span>
              ) : (
                MESSAGES.chat.window.actions.verifyIdentity
              )}
            </button>
          )}
        </div>

        <div
          ref={scrollContainerRef}
          className="flex-1 overflow-y-auto px-4 py-4 space-y-3 scrollbar-custom relative chat-scroll-area"
        >
          {isLoadingFingerprint && (
            <div className="flex items-center justify-center py-8">
              <div className="flex flex-col items-center gap-3">
                <Spinner size="lg" borderColorClass="border-emerald-400" />
                <p className="text-xs text-emerald-500/80">
                  {MESSAGES.chat.window.labels.securityCheck}
                </p>
              </div>
            </div>
          )}

          {!isLoadingFingerprint && isLoading && (
            <div className="flex items-center justify-center py-8">
              <div className="flex flex-col items-center gap-3">
                <Spinner size="lg" borderColorClass="border-emerald-400" />
                <p className="text-xs text-emerald-500/80">{stateMessage}</p>
              </div>
            </div>
          )}

          {!isLoading && messages.length === 0 && isSessionActive && (
            <div className="flex flex-col items-center justify-center py-12 px-4 animate-[fadeIn_0.4s_ease-out]">
              <div className="w-16 h-16 rounded-full bg-emerald-900/30 border-2 border-emerald-700/40 flex items-center justify-center mb-4 animate-[scaleIn_0.3s_ease-out]">
                <svg
                  className="w-8 h-8 text-emerald-400/70"
                  fill="none"
                  stroke="currentColor"
                  viewBox="0 0 24 24"
                >
                  <path
                    strokeLinecap="round"
                    strokeLinejoin="round"
                    strokeWidth={1.5}
                    d="M8 12h.01M12 12h.01M16 12h.01M21 12c0 4.418-4.03 8-9 8a9.863 9.863 0 01-4.255-.949L3 20l1.395-3.72C3.512 15.042 3 13.574 3 12c0-4.418 4.03-8 9-8s9 3.582 9 8z"
                  />
                </svg>
              </div>
              <p className="text-sm font-medium text-emerald-300 mb-1">
                {MESSAGES.chat.window.emptyState.title}
              </p>
              <p className="text-xs text-emerald-500/70 text-center max-w-xs">
                {MESSAGES.chat.window.emptyState.subtitle}
              </p>
            </div>
          )}

          <MessageList
            messages={messages}
            myUserId={myUserId}
            peerUsername={peer.username}
            peerActivity={peerActivity}
            isSessionActive={isSessionActive}
            isChatBlocked={isChatBlocked}
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
            scrollElementRef={scrollContainerRef}
          />

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
                      {MESSAGES.chat.window.banners.securityCodesChanged.title}
                    </p>
                    <p className="text-xs text-yellow-500/80 mt-1">
                      {
                        MESSAGES.chat.window.banners.securityCodesChanged
                          .description
                      }
                    </p>
                  </div>
                </div>
              </div>
            </div>
          )}

          {error && state === 'error' && (
            <div className="flex items-center justify-center py-4 animate-[fadeIn_0.3s_ease-out]">
              <div className="bg-red-900/30 border border-red-600/60 rounded-lg px-4 py-2.5 max-w-md">
                <p className="text-sm text-red-300 text-center">{error}</p>
              </div>
            </div>
          )}
        </div>

        <form
          onSubmit={handleSend}
          className="border-t border-emerald-700/60 bg-black/80 px-4 py-3"
        >
          {replyTo && (
            <div className="mb-2 rounded-lg border border-emerald-700/60 bg-black/70 px-3 py-2 flex items-start justify-between gap-2">
              <div className="flex-1 min-w-0">
                <p className="text-[11px] text-emerald-400/80">
                  {MESSAGES.chat.window.labels.replyToMessage}
                </p>
                <p className="text-xs text-emerald-50 truncate">
                  {replyTo.text
                    ? replyTo.text
                    : replyTo.voice
                      ? MESSAGES.chat.window.reply.voice
                      : replyTo.file
                        ? MESSAGES.chat.window.reply.file
                        : MESSAGES.chat.window.reply.message}
                </p>
              </div>
              <button
                type="button"
                onClick={() => setReplyTo(null)}
                className="flex-shrink-0 w-5 h-5 rounded-full bg-emerald-800/60 hover:bg-emerald-700 text-emerald-50 flex items-center justify-center text-[10px] transition-colors mt-1"
                aria-label={MESSAGES.chat.window.aria.cancelReply}
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
                  showToast(MESSAGES.chat.window.toasts.voiceSent, 'success');
                } catch (err) {
                  const errorMessage = getFriendlyErrorMessage(
                    err,
                    MESSAGES.chat.window.errors.failedToSendVoice
                  );
                  showToast(errorMessage, 'error');
                } finally {
                  sendTyping(null);
                }
              }}
              onError={(error) => showToast(error, 'error')}
              disabled={!isSessionActive || isChatBlocked}
              onRecordingChange={useCallback((isRecording: boolean) => {
                setIsVoiceRecording(isRecording);
              }, [])}
            />
            <VideoRecorder
              onRecorded={async (file, duration) => {
                if (!isSessionActive || isChatBlocked) return;
                try {
                  await sendFile(file, 'both', undefined, duration);
                  showToast(MESSAGES.chat.window.toasts.videoSent, 'success');
                } catch (err) {
                  const errorMessage = getFriendlyErrorMessage(
                    err,
                    MESSAGES.chat.window.errors.failedToSendVideo
                  );
                  showToast(errorMessage, 'error');
                }
              }}
              onError={(error) => showToast(error, 'error')}
              disabled={!isSessionActive || isChatBlocked || isVoiceRecording}
            />
            <div className="flex-1 relative flex items-center">
              <textarea
                ref={inputRef}
                value={messageText}
                onChange={handleMessageTextChange}
                onKeyDown={handleKeyDown}
                onPaste={handlePaste}
                onBlur={handleInputBlur}
                disabled={
                  !isSessionActive ||
                  isSending ||
                  isChatBlocked ||
                  isVoiceRecording
                }
                placeholder={
                  isChatBlocked
                    ? MESSAGES.chat.window.placeholders.blocked
                    : isSessionActive
                      ? MESSAGES.chat.window.placeholders.message
                      : MESSAGES.chat.window.placeholders.waitingForSession
                }
                rows={1}
                className="w-full resize-none rounded-md bg-black border border-emerald-700 px-3 pr-10 py-2.5 text-sm text-emerald-50 placeholder-emerald-500/50 outline-none focus:ring-2 focus:ring-emerald-500 disabled:opacity-50 disabled:cursor-not-allowed transition-all scrollbar-custom"
                style={{
                  maxHeight: '120px',
                  minHeight: `${INPUT_MIN_HEIGHT_PX}px`,
                  height: `${INPUT_MIN_HEIGHT_PX}px`,
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
                title={MESSAGES.chat.window.actions.attachFileTitle}
              >
                {isSendingFile ? (
                  <Spinner size="xs" borderColorClass="border-emerald-300" />
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
              disabled={
                !messageText.trim() ||
                !isSessionActive ||
                isSending ||
                isVoiceRecording
              }
              className="rounded-md bg-emerald-500 hover:bg-emerald-400 disabled:bg-emerald-700 disabled:cursor-not-allowed text-sm font-medium px-4 h-10 text-black smooth-transition button-press glow-emerald-hover flex items-center justify-center min-w-[80px]"
            >
              {isSending ? (
                <Spinner size="sm" borderColorClass="border-black" />
              ) : (
                MESSAGES.chat.window.actions.send
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
            showToast(MESSAGES.chat.security.identityVerifiedToast, 'success');
          }}
        />
      )}

      {showAccessDialog && pendingFile && (
        <FileAccessDialog
          filename={pendingFile.name}
          onSelect={async (mode: FileAccessMode) => {
            if (!pendingFile) {
              setShowAccessDialog(false);
              return;
            }

            const validationError = validateFileSize(pendingFile, {
              fileType: 'file',
            });
            if (validationError) {
              showToast(getFileValidationError(validationError)!, 'error');
              setShowAccessDialog(false);
              setPendingFile(null);
              return;
            }

            setShowAccessDialog(false);
            setIsSendingFile(true);
            try {
              await sendFile(pendingFile, mode);
              showToast(MESSAGES.chat.window.toasts.fileSent, 'success');
            } catch (err) {
              const errorMessage = getFriendlyErrorMessage(
                err,
                MESSAGES.chat.window.errors.failedToSendFile
              );
              showToast(errorMessage, 'error');
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
