export function normalizeFingerprint(fingerprint: string): string {
  return fingerprint.replace(/[^0-9a-fA-F]/g, '').toLowerCase();
}

export function formatFingerprint(fingerprint: string): string {
  const normalized = normalizeFingerprint(fingerprint);
  return normalized.match(/.{1,4}/g)?.join(' ') || normalized;
}

export function fingerprintToEmojis(fingerprint: string): string {
  const normalized = normalizeFingerprint(fingerprint);
  const emojis = [
    '🐶',
    '🐱',
    '🐭',
    '🐹',
    '🐰',
    '🦊',
    '🐻',
    '🐼',
    '🐨',
    '🐯',
    '🦁',
    '🐮',
    '🐷',
    '🐸',
    '🐵',
    '🙈',
  ];

  const result: string[] = [];
  for (let i = 0; i < 8 && i * 4 < normalized.length; i++) {
    const hex = normalized.slice(i * 4, i * 4 + 4);
    if (hex.length === 4) {
      const value = parseInt(hex, 16);
      result.push(emojis[value % emojis.length]);
    }
  }

  return result.join(' ');
}

const VERIFIED_PEERS_STORAGE = 'verified_peers';
const FINGERPRINT_HISTORY_STORAGE = 'fingerprint_history';

interface FingerprintHistory {
  fingerprint: string;
  verifiedAt: number;
  changedAt?: number;
}

function getVerifiedPeers(): Record<string, string> {
  const stored = localStorage.getItem(VERIFIED_PEERS_STORAGE);
  if (!stored) return {};

  try {
    return JSON.parse(stored) as Record<string, string>;
  } catch {
    return {};
  }
}

export function saveVerifiedPeer(userId: string, fingerprint: string): void {
  const normalized = normalizeFingerprint(fingerprint);
  const peers = getVerifiedPeers();
  const oldFingerprint = peers[userId];
  const now = Date.now();

  peers[userId] = normalized;
  localStorage.setItem(VERIFIED_PEERS_STORAGE, JSON.stringify(peers));

  const history = getFingerprintHistory();
  if (!history[userId]) {
    history[userId] = [];
  }

  const existingEntry = history[userId].find(
    (h) => h.fingerprint === normalized,
  );
  if (existingEntry) {
    if (!existingEntry.verifiedAt) {
      existingEntry.verifiedAt = now;
    }
  } else {
    history[userId].push({
      fingerprint: normalized,
      verifiedAt: now,
    });
  }

  if (oldFingerprint && oldFingerprint !== normalized) {
    const oldEntry = history[userId].find(
      (h) => h.fingerprint === oldFingerprint,
    );
    if (oldEntry && !oldEntry.changedAt) {
      oldEntry.changedAt = now;
    }
  }

  localStorage.setItem(FINGERPRINT_HISTORY_STORAGE, JSON.stringify(history));
}

function getFingerprintHistory(): Record<string, FingerprintHistory[]> {
  const stored = localStorage.getItem(FINGERPRINT_HISTORY_STORAGE);
  if (!stored) return {};

  try {
    return JSON.parse(stored) as Record<string, FingerprintHistory[]>;
  } catch {
    return {};
  }
}

export function isPeerVerified(userId: string, fingerprint: string): boolean {
  const peers = getVerifiedPeers();
  return peers[userId] === normalizeFingerprint(fingerprint);
}

export function hasPeerFingerprintChanged(
  userId: string,
  currentFingerprint: string,
): boolean {
  const peers = getVerifiedPeers();
  const stored = peers[userId];
  return (
    stored !== undefined && stored !== normalizeFingerprint(currentFingerprint)
  );
}

export function getVerifiedPeerFingerprint(userId: string): string | undefined {
  return getVerifiedPeers()[userId];
}

export function saveVerifiedPeerFingerprint(
  userId: string,
  fingerprint: string,
): void {
  saveVerifiedPeer(userId, fingerprint);
}
