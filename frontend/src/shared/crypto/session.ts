export type SessionKey = CryptoKey;

export async function deriveSessionKey(
  privateKey: CryptoKey,
  peerPublicKey: CryptoKey,
): Promise<SessionKey> {
  const sharedSecret = await crypto.subtle.deriveBits(
    {
      name: 'ECDH',
      public: peerPublicKey,
    },
    privateKey,
    256,
  );

  const baseKey = await crypto.subtle.importKey(
    'raw',
    sharedSecret,
    {
      name: 'HKDF',
    },
    false,
    ['deriveKey'],
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
      length: 256,
    },
    true,
    ['encrypt', 'decrypt'],
  );

  return sessionKey;
}
