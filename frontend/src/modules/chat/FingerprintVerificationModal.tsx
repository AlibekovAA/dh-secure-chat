import { useCallback, useEffect, useState } from 'react';
import { getFingerprint } from './api';
import {
  fingerprintToEmojis,
  formatFingerprint,
  hasPeerFingerprintChanged,
  isPeerVerified,
  saveVerifiedPeer,
} from '../../shared/crypto/fingerprint';

type Props = {
  token: string;
  peerId: string;
  peerUsername: string;
  myFingerprint: string | null;
  onClose(): void;
  onVerified?(): void;
};

export function FingerprintVerificationModal({
  token,
  peerId,
  peerUsername,
  myFingerprint,
  onClose,
  onVerified,
}: Props) {
  const [peerFingerprint, setPeerFingerprint] = useState<string | null>(null);
  const [isLoading, setIsLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [isVerified, setIsVerified] = useState(false);

  useEffect(() => {
    const loadFingerprint = async () => {
      try {
        setIsLoading(true);
        setError(null);
        const response = await getFingerprint(peerId, token);
        setPeerFingerprint(response.fingerprint);
        setIsVerified(isPeerVerified(peerId, response.fingerprint));
      } catch (err) {
        setError(
          err instanceof Error
            ? err.message
            : 'Не удалось загрузить fingerprint',
        );
      } finally {
        setIsLoading(false);
      }
    };

    void loadFingerprint();
  }, [token, peerId]);

  const handleVerify = useCallback(() => {
    if (!peerFingerprint) return;
    saveVerifiedPeer(peerId, peerFingerprint);
    setIsVerified(true);
    onVerified?.();
  }, [peerId, peerFingerprint, onVerified]);

  const hasChanged = peerFingerprint
    ? hasPeerFingerprintChanged(peerId, peerFingerprint)
    : false;

  return (
    <div
      className="fixed inset-0 z-[60] flex items-center justify-center bg-black/90 backdrop-blur-sm"
      onClick={onClose}
      style={{ willChange: 'opacity' }}
    >
      <div
        className="w-full max-w-lg mx-4 bg-black border border-emerald-700 rounded-xl overflow-hidden animate-[fadeIn_0.3s_ease-out,scaleIn_0.3s_ease-out] shadow-2xl shadow-emerald-900/30"
        onClick={(e) => e.stopPropagation()}
        style={{ willChange: 'transform, opacity' }}
      >
        <div className="px-6 py-4 border-b border-emerald-700/60 bg-black/80">
          <div className="flex items-center justify-between">
            <h2 className="text-lg font-semibold text-emerald-300">
              Верификация Identity
            </h2>
            <button
              type="button"
              onClick={onClose}
              className="text-emerald-400 hover:text-emerald-200 transition-colors"
              aria-label="Закрыть"
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

        <div className="px-6 py-6 space-y-6">
          {isLoading && (
            <div className="flex items-center justify-center py-8">
              <div className="flex flex-col items-center gap-3">
                <div className="w-8 h-8 border-2 border-emerald-400 border-t-transparent rounded-full animate-spin" />
                <p className="text-xs text-emerald-500/80">
                  Загрузка fingerprint...
                </p>
              </div>
            </div>
          )}

          {error && (
            <div className="bg-red-900/20 border border-red-700/40 rounded-lg px-4 py-3">
              <p className="text-sm text-red-400">{error}</p>
            </div>
          )}

          {!isLoading && !error && peerFingerprint && (
            <>
              {hasChanged && (
                <div className="bg-yellow-900/20 border border-yellow-700/40 rounded-lg px-4 py-3">
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
                        Внимание: Fingerprint изменился!
                      </p>
                      <p className="text-xs text-yellow-500/80 mt-1">
                        Fingerprint этого пользователя отличается от
                        сохранённого. Убедитесь, что вы общаетесь с правильным
                        человеком.
                      </p>
                    </div>
                  </div>
                </div>
              )}

              <div className="space-y-4">
                <div>
                  <p className="text-xs font-medium text-emerald-400 mb-2">
                    Ваш Fingerprint
                  </p>
                  <div className="bg-emerald-900/20 border border-emerald-700/40 rounded-lg px-4 py-3 space-y-2">
                    <p className="text-xs font-mono text-emerald-200 break-all">
                      {myFingerprint
                        ? formatFingerprint(myFingerprint)
                        : 'Не загружен'}
                    </p>
                    {myFingerprint && (
                      <div className="flex items-center gap-2 pt-2 border-t border-emerald-700/30">
                        <span className="text-xs text-emerald-400/80">
                          Visual:
                        </span>
                        <span className="text-lg">
                          {fingerprintToEmojis(myFingerprint)}
                        </span>
                      </div>
                    )}
                  </div>
                </div>

                <div>
                  <p className="text-xs font-medium text-emerald-400 mb-2">
                    Fingerprint {peerUsername}
                  </p>
                  <div className="bg-emerald-900/20 border border-emerald-700/40 rounded-lg px-4 py-3 space-y-2">
                    <p className="text-xs font-mono text-emerald-200 break-all">
                      {formatFingerprint(peerFingerprint)}
                    </p>
                    <div className="flex items-center gap-2 pt-2 border-t border-emerald-700/30">
                      <span className="text-xs text-emerald-400/80">
                        Visual:
                      </span>
                      <span className="text-lg">
                        {fingerprintToEmojis(peerFingerprint)}
                      </span>
                    </div>
                  </div>
                </div>
              </div>

              <div className="bg-emerald-900/10 border border-emerald-700/30 rounded-lg px-4 py-3">
                <p className="text-xs text-emerald-400/90 leading-relaxed mb-2">
                  <strong className="text-emerald-300">Инструкция:</strong>{' '}
                  Сравните эти fingerprint'ы вне приложения (например, по
                  телефону или в другом канале связи). Если они совпадают,
                  нажмите "Подтвердить". Это защитит вас от атак типа
                  man-in-the-middle.
                </p>
                <p className="text-xs text-emerald-500/70">
                  💡 <strong>Совет:</strong> Используйте визуальные коды
                  (эмодзи) для быстрого сравнения по телефону — они легче
                  запоминаются и произносятся.
                </p>
              </div>

              <div
                className={`bg-emerald-900/20 border border-emerald-500/40 rounded-lg px-4 py-3 transition-opacity duration-300 min-h-[3.5rem] ${
                  isVerified ? 'opacity-100' : 'opacity-0 pointer-events-none'
                }`}
              >
                <div className="flex items-center gap-2">
                  <svg
                    className="w-5 h-5 text-emerald-400"
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
                  <p className="text-sm text-emerald-300">
                    Identity подтверждён
                  </p>
                </div>
              </div>
            </>
          )}
        </div>

        <div className="px-6 py-4 border-t border-emerald-700/60 bg-black/80 flex items-center justify-end gap-3">
          <button
            type="button"
            onClick={onClose}
            className="px-4 py-2 text-sm font-medium text-emerald-400 hover:text-emerald-200 transition-colors"
          >
            Закрыть
          </button>
          {!isLoading && !error && peerFingerprint && !isVerified && (
            <button
              type="button"
              onClick={handleVerify}
              className="px-4 py-2 text-sm font-medium bg-emerald-500 hover:bg-emerald-400 text-black rounded-md transition-colors"
            >
              Подтвердить
            </button>
          )}
        </div>
      </div>
    </div>
  );
}
