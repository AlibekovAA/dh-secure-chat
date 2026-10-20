import { checkWebCryptoSupport as checkBrowserSupport } from '@/shared/browser-support';
import { IDENTITY_KEY_STORAGE, MASTER_KEY_STORAGE } from '@/shared/constants';
import { MESSAGES } from '@/shared/messages';

export type IdentityKeyPair = {
  publicKey: CryptoKey;
  privateKey: CryptoKey;
};

function checkWebCryptoSupport(): void {
  checkBrowserSupport();
}

export async function generateIdentityKeyPair(): Promise<IdentityKeyPair> {
  checkWebCryptoSupport();

  try {
    const keyPair = await crypto.subtle.generateKey(
      {
        name: 'ECDSA',
        namedCurve: 'P-256',
      },
      false,
      ['sign', 'verify']
    );

    return {
      publicKey: keyPair.publicKey,
      privateKey: keyPair.privateKey,
    };
  } catch (err) {
    const error = err instanceof Error ? err.message : String(err);
    if (error.includes('not supported') || error.includes('not implemented')) {
      throw new Error(MESSAGES.crypto.identity.errors.keygenNotSupported);
    }
    if (error.includes('secure context') || error.includes('HTTPS')) {
      throw new Error(MESSAGES.crypto.identity.errors.keygenRequiresHttps);
    }
    throw new Error(MESSAGES.crypto.identity.errors.keygenFailed(error));
  }
}

export async function exportPublicKey(publicKey: CryptoKey): Promise<string> {
  const exported = await crypto.subtle.exportKey('spki', publicKey);
  const base64 = btoa(String.fromCharCode(...new Uint8Array(exported)));
  return base64;
}

export async function importPublicKey(base64: string): Promise<CryptoKey> {
  const binary = Uint8Array.from(atob(base64), (c) => c.charCodeAt(0));
  return await crypto.subtle.importKey(
    'spki',
    binary,
    {
      name: 'ECDH',
      namedCurve: 'P-256',
    },
    true,
    []
  );
}

function clearLegacyIdentityStorage(): void {
  try {
    localStorage.removeItem(IDENTITY_KEY_STORAGE);
    localStorage.removeItem(MASTER_KEY_STORAGE);
  } catch {
    void 0;
  }
}

function isStoredIdentityPrivateKey(value: unknown): value is CryptoKey {
  if (typeof CryptoKey === 'undefined' || !(value instanceof CryptoKey))
    return false;
  const algorithm = value.algorithm as EcKeyAlgorithm;
  return (
    value.type === 'private' &&
    !value.extractable &&
    algorithm.name === 'ECDSA' &&
    algorithm.namedCurve === 'P-256' &&
    value.usages.includes('sign')
  );
}

export async function saveIdentityPrivateKey(
  privateKey: CryptoKey
): Promise<void> {
  clearLegacyIdentityStorage();

  if (!isStoredIdentityPrivateKey(privateKey)) {
    throw new Error(
      MESSAGES.crypto.identity.errors.keygenFailed('invalid private key')
    );
  }

  const { saveKey } = await import('@/shared/storage/indexeddb');
  await saveKey(IDENTITY_KEY_STORAGE, privateKey, 'identity');
}

export async function loadIdentityPrivateKey(): Promise<CryptoKey | null> {
  clearLegacyIdentityStorage();

  try {
    const { deleteKey, loadKey } = await import('@/shared/storage/indexeddb');
    const stored = await loadKey<CryptoKey>(IDENTITY_KEY_STORAGE);
    if (isStoredIdentityPrivateKey(stored)) {
      return stored;
    }
    if (stored !== null) {
      await deleteKey(IDENTITY_KEY_STORAGE);
    }
  } catch {
    return null;
  }

  return null;
}
