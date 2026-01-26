import {
  encryptBinary,
  decryptBinary,
} from '@/shared/crypto/binary-encryption';
import type { SessionKey } from '@/shared/crypto/session';

export async function encrypt(
  sessionKey: SessionKey,
  plaintext: string
): Promise<{ ciphertext: string; nonce: string }> {
  const encoded = new TextEncoder().encode(plaintext);
  return await encryptBinary(sessionKey, encoded);
}

export async function decrypt(
  sessionKey: SessionKey,
  ciphertext: string,
  nonce: string
): Promise<string> {
  const decrypted = await decryptBinary(sessionKey, ciphertext, nonce);
  return new TextDecoder().decode(decrypted);
}
