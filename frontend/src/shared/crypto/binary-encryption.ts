import type { SessionKey } from './session';
import { BASE64_CHUNK_SIZE } from '../constants';

async function encryptBinaryInternal(
  sessionKey: CryptoKey,
  data: Uint8Array,
): Promise<{ ciphertext: string; nonce: string }> {
  const nonce = crypto.getRandomValues(new Uint8Array(12));
  const buffer = new Uint8Array(data).buffer;

  const ciphertext = await crypto.subtle.encrypt(
    {
      name: 'AES-GCM',
      iv: nonce,
    },
    sessionKey,
    buffer,
  );

  const ciphertextArray = new Uint8Array(ciphertext);
  let ciphertextString = '';
  for (let i = 0; i < ciphertextArray.length; i += BASE64_CHUNK_SIZE) {
    const chunk = ciphertextArray.slice(i, i + BASE64_CHUNK_SIZE);
    ciphertextString += String.fromCharCode.apply(null, Array.from(chunk));
  }
  const ciphertextBase64 = btoa(ciphertextString);

  const nonceBase64 = btoa(String.fromCharCode.apply(null, Array.from(nonce)));

  return {
    ciphertext: ciphertextBase64,
    nonce: nonceBase64,
  };
}

async function decryptBinaryInternal(
  sessionKey: CryptoKey,
  ciphertext: string,
  nonce: string,
): Promise<Uint8Array> {
  const ciphertextBinary = Uint8Array.from(atob(ciphertext), (c) =>
    c.charCodeAt(0),
  );
  const nonceBinary = Uint8Array.from(atob(nonce), (c) => c.charCodeAt(0));

  const plaintext = await crypto.subtle.decrypt(
    {
      name: 'AES-GCM',
      iv: nonceBinary,
    },
    sessionKey,
    ciphertextBinary,
  );

  return new Uint8Array(plaintext);
}

export async function encryptBinary(
  sessionKey: SessionKey,
  data: Uint8Array,
): Promise<{ ciphertext: string; nonce: string }> {
  return encryptBinaryInternal(sessionKey as CryptoKey, data);
}

export async function decryptBinary(
  sessionKey: SessionKey,
  ciphertext: string,
  nonce: string,
): Promise<Uint8Array> {
  return decryptBinaryInternal(sessionKey as CryptoKey, ciphertext, nonce);
}

export {
  encryptBinaryInternal as encryptBinaryWithKey,
  decryptBinaryInternal as decryptBinaryWithKey,
};
