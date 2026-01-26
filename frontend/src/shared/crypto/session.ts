import { AES_KEY_SIZE } from '@/shared/constants';

export type SessionKey = CryptoKey;

export async function deriveSessionKey(
  privateKey: CryptoKey,
  peerPublicKey: CryptoKey
): Promise<SessionKey> {
  const sharedSecret = await crypto.subtle.deriveBits(
    {
      name: 'ECDH',
      public: peerPublicKey,
    },
    privateKey,
    AES_KEY_SIZE
  );

  const baseKey = await crypto.subtle.importKey(
    'raw',
    sharedSecret,
    {
      name: 'HKDF',
    },
    false,
    ['deriveKey']
  );

  const sessionKey = await crypto.subtle.deriveKey(
    {
      name: 'HKDF',
      hash: 'SHA-256',
      salt: new Uint8Array(0),
      info: new TextEncoder().encode('dh-secure-chat-session-key'),
    },
    baseKey,
    {
      name: 'AES-GCM',
      length: AES_KEY_SIZE,
    },
    true,
    ['encrypt', 'decrypt']
  );

  return sessionKey;
}
