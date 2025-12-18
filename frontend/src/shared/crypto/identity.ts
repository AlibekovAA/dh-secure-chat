export type IdentityKeyPair = {
  publicKey: CryptoKey;
  privateKey: CryptoKey;
};

function checkWebCryptoSupport(): void {
  if (!crypto || !crypto.subtle) {
    throw new Error(
      'Web Crypto API не поддерживается в этом браузере. Используйте современный браузер (Chrome, Firefox, Safari, Edge).',
    );
  }
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
      ['deriveKey', 'deriveBits'],
    );

    return {
      publicKey: keyPair.publicKey,
      privateKey: keyPair.privateKey,
    };
  } catch (err) {
    const error = err instanceof Error ? err.message : String(err);
    if (error.includes('not supported') || error.includes('not implemented')) {
      throw new Error(
        'Генерация ключей не поддерживается. Используйте современный браузер (Chrome, Firefox, Safari, Edge).',
      );
    }
    if (error.includes('secure context') || error.includes('HTTPS')) {
      throw new Error(
        'Для генерации ключей требуется безопасное соединение (HTTPS).',
      );
    }
    throw new Error(`Ошибка генерации ключей: ${error}`);
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
    [],
  );
}

const IDENTITY_KEY_STORAGE = 'identity_private_key';

export function saveIdentityPrivateKey(privateKey: CryptoKey): Promise<void> {
  return crypto.subtle.exportKey('pkcs8', privateKey).then((exported) => {
    const base64 = btoa(String.fromCharCode(...new Uint8Array(exported)));
    localStorage.setItem(IDENTITY_KEY_STORAGE, base64);
  });
}

export async function loadIdentityPrivateKey(): Promise<CryptoKey | null> {
  const stored = localStorage.getItem(IDENTITY_KEY_STORAGE);
  if (!stored) {
    return null;
  }

  try {
    const binary = Uint8Array.from(atob(stored), (c) => c.charCodeAt(0));
    return await crypto.subtle.importKey(
      'pkcs8',
      binary,
      {
        name: 'ECDH',
        namedCurve: 'P-256',
      },
      true,
      ['deriveKey', 'deriveBits'],
    );
  } catch {
    return null;
  }
}
