import { encryptBinary, decryptBinary } from './binary-encryption';
import type { SessionKey } from './session';

const CHUNK_SIZE = 1024 * 1024;

export interface EncryptedChunk {
  ciphertext: string;
  nonce: string;
}

export async function encryptFile(
  sessionKey: SessionKey,
  file: File,
): Promise<{ chunks: EncryptedChunk[]; totalSize: number }> {
  const arrayBuffer = await file.arrayBuffer();
  const totalSize = arrayBuffer.byteLength;
  const chunks: EncryptedChunk[] = [];

  for (let offset = 0; offset < totalSize; offset += CHUNK_SIZE) {
    const chunkEnd = Math.min(offset + CHUNK_SIZE, totalSize);
    const chunk = arrayBuffer.slice(offset, chunkEnd);
    const chunkBytes = new Uint8Array(chunk);

    const encrypted = await encryptBinary(sessionKey, chunkBytes);
    chunks.push(encrypted);
  }

  return { chunks, totalSize };
}

export async function decryptFile(
  sessionKey: SessionKey,
  chunks: EncryptedChunk[],
): Promise<Blob> {
  const decryptedChunks: Uint8Array[] = [];

  for (const chunk of chunks) {
    const decrypted = await decryptBinary(
      sessionKey,
      chunk.ciphertext,
      chunk.nonce,
    );
    decryptedChunks.push(decrypted);
  }

  const totalLength = decryptedChunks.reduce(
    (sum, chunk) => sum + chunk.length,
    0,
  );
  const result = new Uint8Array(totalLength);
  let offset = 0;

  for (const chunk of decryptedChunks) {
    result.set(chunk, offset);
    offset += chunk.length;
  }

  return new Blob([result]);
}

export function calculateChunks(fileSize: number): number {
  return Math.ceil(fileSize / CHUNK_SIZE);
}

export function getChunkSize(): number {
  return CHUNK_SIZE;
}
