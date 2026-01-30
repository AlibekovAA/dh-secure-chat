import { useEffect, useMemo, useState } from 'react';
import { getFingerprint } from '@/modules/chat/api';
import { fingerprintToEmojis, formatFingerprint } from '@/shared/crypto/fingerprint';
import { Spinner } from '@/shared/ui/Spinner';
import { MESSAGES } from '@/shared/messages';

type Props = {
  userId: string;
  onClose(): void;
};

export function MyFingerprintModal({ userId, onClose }: Props) {
  const [fingerprint, setFingerprint] = useState<string | null>(null);
  const [isLoading, setIsLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    const load = async () => {
      try {
        setIsLoading(true);
        setError(null);
        const response = await getFingerprint(userId);
        setFingerprint(response.fingerprint);
      } catch (err) {
        setError(
          err instanceof Error
            ? err.message
            : MESSAGES.chat.fingerprintModal.errors.failedToLoadFingerprint
        );
      } finally {
        setIsLoading(false);
      }
    };

    void load();
  }, [userId]);

  const formatted = useMemo(
    () => (fingerprint ? formatFingerprint(fingerprint) : null),
    [fingerprint]
  );

  const emojis = useMemo(
    () => (fingerprint ? fingerprintToEmojis(fingerprint) : null),
    [fingerprint]
  );

  return (
    <div
      className="fixed inset-0 z-[60] flex items-center justify-center bg-black/80 backdrop-blur-sm animate-[backdropEnter_0.2s_ease-out]"
      onClick={onClose}
      style={{ willChange: 'opacity' }}
    >
      <div
        className="w-full max-w-lg mx-4 bg-black border border-emerald-700 rounded-xl overflow-hidden shadow-lg shadow-emerald-900/20 animate-[modalEnter_0.2s_ease-out]"
        onClick={(e) => e.stopPropagation()}
        style={{ willChange: 'transform, opacity' }}
      >
        <div className="px-6 py-4 border-b border-emerald-700/60 bg-black/80">
          <div className="flex items-center justify-between">
            <h2 className="text-lg font-semibold text-emerald-300">
              {MESSAGES.chat.fingerprintModal.myFingerprintModal.title}
            </h2>
            <button
              type="button"
              onClick={onClose}
              className="text-emerald-400 hover:text-emerald-200 transition-colors"
              aria-label={MESSAGES.chat.fingerprintModal.aria.close}
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
                  d="M6 18L18 6M6 6l12 12"
                />
              </svg>
            </button>
          </div>
        </div>

        <div className="px-6 py-6">
          {isLoading && (
            <div className="flex flex-col items-center justify-center py-8 gap-3">
              <Spinner size="lg" borderColorClass="border-emerald-400" />
              <p className="text-xs text-emerald-500/80">
                {MESSAGES.chat.fingerprintModal.states.loadingFingerprint}
              </p>
            </div>
          )}

          {error && (
            <div className="bg-red-900/20 border border-red-700/40 rounded-lg px-4 py-3">
              <p className="text-sm text-red-400">{error}</p>
            </div>
          )}

          {!isLoading && !error && fingerprint && (
            <div className="space-y-4">
              <div>
                <p className="text-xs font-medium text-emerald-400 mb-2">
                  {MESSAGES.chat.fingerprintModal.labels.myFingerprint}
                </p>
                <div className="bg-emerald-900/20 border border-emerald-700/40 rounded-lg px-4 py-3 space-y-2">
                  <p className="text-xs font-mono text-emerald-200 break-all">
                    {formatted}
                  </p>
                  {emojis && (
                    <div className="flex items-center gap-2 pt-2 border-t border-emerald-700/30">
                      <span className="text-xs text-emerald-400/80">
                        {MESSAGES.chat.fingerprintModal.labels.visually}
                      </span>
                      <span className="text-lg">{emojis}</span>
                    </div>
                  )}
                </div>
              </div>
              <p className="text-xs text-emerald-500/70">
                {MESSAGES.chat.fingerprintModal.info.tip.emoji}{' '}
                <strong className="text-emerald-400/90">
                  {MESSAGES.chat.fingerprintModal.info.tip.title}
                </strong>{' '}
                {MESSAGES.chat.fingerprintModal.info.tip.text}
              </p>
            </div>
          )}
        </div>

        <div className="px-6 py-4 border-t border-emerald-700/60 bg-black/80 flex justify-end">
          <button
            type="button"
            onClick={onClose}
            className="px-4 py-2 text-sm font-medium text-emerald-400 hover:text-emerald-200 transition-colors"
          >
            {MESSAGES.chat.fingerprintModal.myFingerprintModal.close}
          </button>
        </div>
      </div>
    </div>
  );
}
