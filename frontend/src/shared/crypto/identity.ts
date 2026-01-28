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
        name: 'ECDH',
        namedCurve: 'P-256',
      },
      true,
      ['deriveKey', 'deriveBits']
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

async function getOrCreateMasterKey(): Promise<CryptoKey> {
  const stored = sessionStorage.getItem(MASTER_KEY_STORAGE);
  if (stored) {
    const keyData = Uint8Array.from(atob(stored), (c) => c.charCodeAt(0));
    return await crypto.subtle.importKey(
      'raw',
      keyData,
      { name: 'AES-GCM' },
      false,
      ['encrypt', 'decrypt']
    );
  }

  const masterKey = crypto.getRandomValues(new Uint8Array(32));
  const key = await crypto.subtle.importKey(
    'raw',
    masterKey,
    { name: 'AES-GCM' },
    false,
    ['encrypt', 'decrypt']
  );

  const exported = await crypto.subtle.exportKey('raw', key);
  sessionStorage.setItem(
    MASTER_KEY_STORAGE,
    btoa(String.fromCharCode(...new Uint8Array(exported)))
  );
  return key;
}

async function encryptKeyData(data: string): Promise<string> {
  try {
    const masterKey = await getOrCreateMasterKey();
    const dataBytes = Uint8Array.from(atob(data), (c) => c.charCodeAt(0));
    const nonce = crypto.getRandomValues(new Uint8Array(12));

    const encrypted = await crypto.subtle.encrypt(
      { name: 'AES-GCM', iv: nonce },
      masterKey,
      dataBytes
    );

    const encryptedArray = new Uint8Array(encrypted);
    const combined = new Uint8Array(nonce.length + encryptedArray.length);
    combined.set(nonce, 0);
    combined.set(encryptedArray, nonce.length);

    return btoa(String.fromCharCode(...combined));
  } catch {
    return data;
  }
}

async function decryptKeyData(encryptedData: string): Promise<string | null> {
  try {
    const masterKey = await getOrCreateMasterKey();
    const combined = Uint8Array.from(atob(encryptedData), (c) =>
      c.charCodeAt(0)
    );

    if (combined.length < 13) {
      return null;
    }

    const nonce = combined.slice(0, 12);
    const encrypted = combined.slice(12);

    const decrypted = await crypto.subtle.decrypt(
      { name: 'AES-GCM', iv: nonce },
      masterKey,
      encrypted
    );

    return btoa(String.fromCharCode(...new Uint8Array(decrypted)));
  } catch {
    return null;
  }
}

async function saveToIndexedDB(key: string, data: string): Promise<void> {
  try {
    if ('indexedDB' in window) {
      const { saveKey } = await import('@/shared/storage/indexeddb');
      await saveKey(key, data, 'identity');
      return;
    }
  } catch {
    void 0;
  }
  localStorage.setItem(key, data);
}

async function loadFromIndexedDB(key: string): Promise<string | null> {
  try {
    if ('indexedDB' in window) {
      const { loadKey } = await import('@/shared/storage/indexeddb');
      const stored = await loadKey(key);
      if (stored) {
        return stored;
      }
    }
  } catch {
    void 0;
  }
  return localStorage.getItem(key);
}

export async function saveIdentityPrivateKey(
  privateKey: CryptoKey
): Promise<void> {
  const exported = await crypto.subtle.exportKey('pkcs8', privateKey);
  const base64 = btoa(String.fromCharCode(...new Uint8Array(exported)));
  const encrypted = await encryptKeyData(base64);
  await saveToIndexedDB(IDENTITY_KEY_STORAGE, encrypted);
}

export async function loadIdentityPrivateKey(): Promise<CryptoKey | null> {
  const stored = await loadFromIndexedDB(IDENTITY_KEY_STORAGE);
  if (!stored) {
    return null;
  }

  try {
    let decrypted: string | null = null;

    try {
      decrypted = await decryptKeyData(stored);
    } catch {
      void 0;
    }

    if (!decrypted) {
      decrypted = stored;
    }

    const binary = Uint8Array.from(atob(decrypted), (c) => c.charCodeAt(0));
    return await crypto.subtle.importKey(
      'pkcs8',
      binary,
      {
        name: 'ECDH',
        namedCurve: 'P-256',
      },
      true,
      ['deriveKey', 'deriveBits']
    );
  } catch (err) {
    return null;
  }
}
