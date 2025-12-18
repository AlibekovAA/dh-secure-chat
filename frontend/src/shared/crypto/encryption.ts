import { encryptBinary, decryptBinary } from './binary-encryption';
import type { SessionKey } from './session';

export async function encrypt(
  sessionKey: SessionKey,
  plaintext: string,
): Promise<{ ciphertext: string; nonce: string }> {
  const encoded = new TextEncoder().encode(plaintext);
  return await encryptBinary(sessionKey, encoded);
}

export async function decrypt(
  sessionKey: SessionKey,
  ciphertext: string,
  nonce: string,
): Promise<string> {
  const decrypted = await decryptBinary(sessionKey, ciphertext, nonce);
  return new TextDecoder().decode(decrypted);
}
