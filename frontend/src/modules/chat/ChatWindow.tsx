import { useCallback, useEffect, useMemo, useRef, useState } from 'react';
import { useChatSession } from './useChatSession';
import type { UserSummary } from './api';

type Props = {
  token: string;
  peer: UserSummary;
  onClose(): void;
};

export function ChatWindow({ token, peer, onClose }: Props) {
  const [messageText, setMessageText] = useState('');
  const [isSending, setIsSending] = useState(false);
  const messagesEndRef = useRef<HTMLDivElement>(null);
  const inputRef = useRef<HTMLTextAreaElement>(null);

  const { state, messages, error, sendMessage, isSessionActive } = useChatSession({
    token,
    peerId: peer.id,
    peerUsername: peer.username,
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
    if (isSessionActive && inputRef.current) {
      inputRef.current.focus();
    }
  }, [isSessionActive]);

  const handleSend = useCallback(
    async (e: React.FormEvent) => {
      e.preventDefault();
      if (!messageText.trim() || isSending || !isSessionActive) return;

      setIsSending(true);
      try {
        await sendMessage(messageText.trim());
        setMessageText('');
        if (inputRef.current) {
          inputRef.current.focus();
        }
      } finally {
        setIsSending(false);
      }
    },
    [messageText, isSending, isSessionActive, sendMessage],
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

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/80 backdrop-blur-sm">
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
                <p className="text-sm whitespace-pre-wrap break-words">{message.text}</p>
                <p className="text-[10px] text-emerald-500/60 mt-1">
                  {new Date(message.timestamp).toLocaleTimeString('ru-RU', {
                    hour: '2-digit',
                    minute: '2-digit',
                  })}
                </p>
              </div>
            </div>
          ))}

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
          <div className="flex items-end gap-2">
            <textarea
              ref={inputRef}
              value={messageText}
              onChange={(e) => setMessageText(e.target.value)}
              onKeyDown={handleKeyDown}
              disabled={!isSessionActive || isSending}
              placeholder={
                isSessionActive
                  ? 'Введите сообщение... (Enter для отправки)'
                  : 'Ожидание установки сессии...'
              }
              rows={1}
              className="flex-1 resize-none rounded-md bg-black border border-emerald-700 px-3 py-2 text-sm text-emerald-50 placeholder-emerald-500/50 outline-none focus:ring-2 focus:ring-emerald-500 disabled:opacity-50 disabled:cursor-not-allowed transition-all"
              style={{
                maxHeight: '120px',
                minHeight: '40px',
              }}
              onInput={(e) => {
                const target = e.currentTarget;
                target.style.height = 'auto';
                target.style.height = `${Math.min(target.scrollHeight, 120)}px`;
              }}
            />
            <button
              type="submit"
              disabled={!messageText.trim() || !isSessionActive || isSending}
              className="rounded-md bg-emerald-500 hover:bg-emerald-400 disabled:bg-emerald-700 disabled:cursor-not-allowed text-sm font-medium px-4 py-2 text-black transition-colors flex items-center justify-center min-w-[80px]"
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
    </div>
  );
}
