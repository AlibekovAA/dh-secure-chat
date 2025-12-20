export async function generateFingerprint(
  publicKey: Uint8Array,
): Promise<string> {
  const buffer = new Uint8Array(publicKey).buffer;
  const hash = await crypto.subtle.digest('SHA-256', buffer);
  return Array.from(new Uint8Array(hash))
    .map((b) => b.toString(16).padStart(2, '0'))
    .join('');
}

export function formatFingerprint(fingerprint: string): string {
  return fingerprint.match(/.{1,4}/g)?.join(' ') || fingerprint;
}

export function fingerprintToEmojis(fingerprint: string): string {
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
  for (let i = 0; i < 8 && i * 4 < fingerprint.length; i++) {
    const start = i * 4;
    const end = start + 4;
    const hex = fingerprint.slice(start, end);
    if (hex.length === 4) {
      const value = parseInt(hex, 16);
      result.push(emojis[value % emojis.length]);
    }
  }

  return result.join(' ');
}

const VERIFIED_PEERS_STORAGE = 'verified_peers';
const FINGERPRINT_HISTORY_STORAGE = 'fingerprint_history';

export interface FingerprintHistory {
  fingerprint: string;
  verifiedAt: number;
  changedAt?: number;
}

export type VerifiedPeer = {
  userId: string;
  fingerprint: string;
};

export function getVerifiedPeers(): Record<string, string> {
  const stored = localStorage.getItem(VERIFIED_PEERS_STORAGE);
  if (!stored) {
    return {};
  }

  try {
    return JSON.parse(stored) as Record<string, string>;
  } catch {
    return {};
  }
}

export function saveVerifiedPeer(userId: string, fingerprint: string): void {
  const peers = getVerifiedPeers();
  const oldFingerprint = peers[userId];
  const now = Date.now();

  peers[userId] = fingerprint;
  localStorage.setItem(VERIFIED_PEERS_STORAGE, JSON.stringify(peers));

  const history = getFingerprintHistory();
  if (!history[userId]) {
    history[userId] = [];
  }

  const existingEntry = history[userId].find(
    (h) => h.fingerprint === fingerprint,
  );
  if (existingEntry) {
    if (!existingEntry.verifiedAt) {
      existingEntry.verifiedAt = now;
    }
  } else {
    history[userId].push({
      fingerprint,
      verifiedAt: now,
    });
  }

  if (oldFingerprint && oldFingerprint !== fingerprint) {
    const oldEntry = history[userId].find(
      (h) => h.fingerprint === oldFingerprint,
    );
    if (oldEntry && !oldEntry.changedAt) {
      oldEntry.changedAt = now;
    }
  }

  localStorage.setItem(FINGERPRINT_HISTORY_STORAGE, JSON.stringify(history));
}

export function getFingerprintHistory(): Record<string, FingerprintHistory[]> {
  const stored = localStorage.getItem(FINGERPRINT_HISTORY_STORAGE);
  if (!stored) {
    return {};
  }

  try {
    return JSON.parse(stored) as Record<string, FingerprintHistory[]>;
  } catch {
    return {};
  }
}

export function getPeerFingerprintHistory(
  userId: string,
): FingerprintHistory[] {
  const history = getFingerprintHistory();
  return history[userId] || [];
}

export function removeVerifiedPeer(userId: string): void {
  const peers = getVerifiedPeers();
  delete peers[userId];
  localStorage.setItem(VERIFIED_PEERS_STORAGE, JSON.stringify(peers));
}

export function isPeerVerified(userId: string, fingerprint: string): boolean {
  const peers = getVerifiedPeers();
  return peers[userId] === fingerprint;
}

export function hasPeerFingerprintChanged(
  userId: string,
  currentFingerprint: string,
): boolean {
  const peers = getVerifiedPeers();
  const stored = peers[userId];
  return stored !== undefined && stored !== currentFingerprint;
}

export function getVerifiedPeerFingerprint(userId: string): string | undefined {
  const peers = getVerifiedPeers();
  return peers[userId];
}

export function saveVerifiedPeerFingerprint(
  userId: string,
  fingerprint: string,
): void {
  saveVerifiedPeer(userId, fingerprint);
}

export function exportVerifiedPeers(): string {
  const data = localStorage.getItem(VERIFIED_PEERS_STORAGE);
  return btoa(data || '{}');
}

export function importVerifiedPeers(encoded: string): boolean {
  try {
    const data = atob(encoded);
    const parsed = JSON.parse(data);
    if (typeof parsed === 'object' && parsed !== null) {
      localStorage.setItem(VERIFIED_PEERS_STORAGE, data);
      return true;
    }
    return false;
  } catch {
    return false;
  }
}

export function exportFingerprintHistory(): string {
  const data = localStorage.getItem(FINGERPRINT_HISTORY_STORAGE);
  return btoa(data || '{}');
}

export function importFingerprintHistory(encoded: string): boolean {
  try {
    const data = atob(encoded);
    const parsed = JSON.parse(data);
    if (typeof parsed === 'object' && parsed !== null) {
      localStorage.setItem(FINGERPRINT_HISTORY_STORAGE, data);
      return true;
    }
    return false;
  } catch {
    return false;
  }
}
