import { MESSAGES } from '@/shared/messages';

export async function signEphemeralKey(
  ephemeralPublicKeyBase64: string,
  identityPrivateKey: CryptoKey
): Promise<string> {
  const data = Uint8Array.from(atob(ephemeralPublicKeyBase64), (c) =>
    c.charCodeAt(0)
  );

  try {
    const signature = await crypto.subtle.sign(
      {
        name: 'ECDSA',
        hash: 'SHA-256',
      },
      identityPrivateKey,
      data
    );

    return btoa(String.fromCharCode(...new Uint8Array(signature)));
  } catch (err) {
    const errorMessage = err instanceof Error ? err.message : String(err);
    throw new Error(
      MESSAGES.crypto.signature.errors.signEphemeralKeyFailed(errorMessage)
    );
  }
}

export async function verifyEphemeralKeySignature(
  ephemeralPublicKeyBase64: string,
  signature: string,
  identityPublicKeyBase64: string
): Promise<boolean> {
  try {
    const data = Uint8Array.from(atob(ephemeralPublicKeyBase64), (c) =>
      c.charCodeAt(0)
    );

    const signatureBytes = Uint8Array.from(atob(signature), (c) =>
      c.charCodeAt(0)
    );

    const publicKeyData = Uint8Array.from(atob(identityPublicKeyBase64), (c) =>
      c.charCodeAt(0)
    );

    const ecdsaPublicKey = await crypto.subtle.importKey(
      'spki',
      publicKeyData,
      {
        name: 'ECDSA',
        namedCurve: 'P-256',
      },
      false,
      ['verify']
    );

    return await crypto.subtle.verify(
      {
        name: 'ECDSA',
        hash: 'SHA-256',
      },
      ecdsaPublicKey,
      signatureBytes,
      data
    );
  } catch {
    return false;
  }
}
