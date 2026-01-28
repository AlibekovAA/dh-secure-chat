import { useCallback, useEffect, useMemo, useState } from 'react';
import { getFingerprint } from '@/modules/chat/api';
import {
  fingerprintToEmojis,
  formatFingerprint,
  hasPeerFingerprintChanged,
  isPeerVerified,
  saveVerifiedPeer,
} from '@/shared/crypto/fingerprint';
import { Spinner } from '@/shared/ui/Spinner';
import { MESSAGES } from '@/shared/messages';

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
        const response = await getFingerprint(peerId);
        setPeerFingerprint(response.fingerprint);
        setIsVerified(isPeerVerified(peerId, response.fingerprint));
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

    void loadFingerprint();
  }, [token, peerId]);

  const handleVerify = useCallback(() => {
    if (!peerFingerprint) return;
    saveVerifiedPeer(peerId, peerFingerprint);
    setIsVerified(true);
    onVerified?.();
  }, [peerId, peerFingerprint, onVerified]);

  const hasChanged = useMemo(
    () =>
      peerFingerprint
        ? hasPeerFingerprintChanged(peerId, peerFingerprint)
        : false,
    [peerId, peerFingerprint]
  );

  const formattedMyFingerprint = useMemo(
    () => (myFingerprint ? formatFingerprint(myFingerprint) : null),
    [myFingerprint]
  );

  const formattedPeerFingerprint = useMemo(
    () => (peerFingerprint ? formatFingerprint(peerFingerprint) : null),
    [peerFingerprint]
  );

  const myFingerprintEmojis = useMemo(
    () => (myFingerprint ? fingerprintToEmojis(myFingerprint) : null),
    [myFingerprint]
  );

  const peerFingerprintEmojis = useMemo(
    () => (peerFingerprint ? fingerprintToEmojis(peerFingerprint) : null),
    [peerFingerprint]
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
              {MESSAGES.chat.fingerprintModal.title}
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

        <div className="px-6 py-6 space-y-6">
          {isLoading && (
            <div className="flex items-center justify-center py-8">
              <div className="flex flex-col items-center gap-3">
                <Spinner size="lg" borderColorClass="border-emerald-400" />
                <p className="text-xs text-emerald-500/80">
                  {MESSAGES.chat.fingerprintModal.states.loadingFingerprint}
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
                        {MESSAGES.chat.fingerprintModal.warnings.changedTitle}
                      </p>
                      <p className="text-xs text-yellow-500/80 mt-1">
                        {MESSAGES.chat.fingerprintModal.warnings.changedBody}
                      </p>
                    </div>
                  </div>
                </div>
              )}

              <div className="space-y-4">
                <div>
                  <p className="text-xs font-medium text-emerald-400 mb-2">
                    {MESSAGES.chat.fingerprintModal.labels.myFingerprint}
                  </p>
                  <div className="bg-emerald-900/20 border border-emerald-700/40 rounded-lg px-4 py-3 space-y-2">
                    <p className="text-xs font-mono text-emerald-200 break-all">
                      {formattedMyFingerprint ||
                        MESSAGES.chat.fingerprintModal.labels.notLoaded}
                    </p>
                    {myFingerprintEmojis && (
                      <div className="flex items-center gap-2 pt-2 border-t border-emerald-700/30">
                        <span className="text-xs text-emerald-400/80">
                          {MESSAGES.chat.fingerprintModal.labels.visually}
                        </span>
                        <span className="text-lg">{myFingerprintEmojis}</span>
                      </div>
                    )}
                  </div>
                </div>

                <div>
                  <p className="text-xs font-medium text-emerald-400 mb-2">
                    {MESSAGES.chat.fingerprintModal.labels.peerFingerprint(
                      peerUsername
                    )}
                  </p>
                  <div className="bg-emerald-900/20 border border-emerald-700/40 rounded-lg px-4 py-3 space-y-2">
                    <p className="text-xs font-mono text-emerald-200 break-all">
                      {formattedPeerFingerprint}
                    </p>
                    <div className="flex items-center gap-2 pt-2 border-t border-emerald-700/30">
                      <span className="text-xs text-emerald-400/80">
                        {MESSAGES.chat.fingerprintModal.labels.visually}
                      </span>
                      <span className="text-lg">{peerFingerprintEmojis}</span>
                    </div>
                  </div>
                </div>
              </div>

              <div className="bg-emerald-900/10 border border-emerald-700/30 rounded-lg px-4 py-3">
                <p className="text-xs text-emerald-400/90 leading-relaxed mb-2">
                  <strong className="text-emerald-300">
                    {MESSAGES.chat.fingerprintModal.info.instructionTitle}
                  </strong>{' '}
                  {MESSAGES.chat.fingerprintModal.info.instructionText}
                </p>
                <p className="text-xs text-emerald-500/70">
                  {MESSAGES.chat.fingerprintModal.info.tip.emoji}{' '}
                  <strong>
                    {MESSAGES.chat.fingerprintModal.info.tip.title}
                  </strong>{' '}
                  {MESSAGES.chat.fingerprintModal.info.tip.text}
                </p>
              </div>

              <div
                className={`bg-emerald-900/20 border border-emerald-500/40 rounded-lg px-4 py-3 min-h-[3.5rem] transition-all duration-200 ease-out ${
                  isVerified
                    ? 'opacity-100 translate-y-0'
                    : 'opacity-0 translate-y-2 pointer-events-none'
                }`}
                style={{
                  transform: isVerified
                    ? 'translateY(0) translateZ(0)'
                    : 'translateY(8px) translateZ(0)',
                }}
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
                    {MESSAGES.chat.security.identityVerifiedLabel}
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
            {MESSAGES.chat.fingerprintModal.actions.close}
          </button>
          {!isLoading && !error && peerFingerprint && !isVerified && (
            <button
              type="button"
              onClick={handleVerify}
              className="px-4 py-2 text-sm font-medium bg-emerald-500 hover:bg-emerald-400 text-black rounded-md transition-colors"
            >
              {MESSAGES.chat.fingerprintModal.actions.verify}
            </button>
          )}
        </div>
      </div>
    </div>
  );
}
