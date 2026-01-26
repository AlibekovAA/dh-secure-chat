import {
  encryptBinary,
  decryptBinary,
} from '@/shared/crypto/binary-encryption';
import type { SessionKey } from '@/shared/crypto/session';
import {
  encryptFileWithWorker,
  decryptFileWithWorker,
} from '@/shared/crypto/file-encryption-worker';
import { FILE_CHUNK_SIZE, WORKER_THRESHOLD } from '@/shared/constants';

export interface EncryptedChunk {
  ciphertext: string;
  nonce: string;
}

let workerSupported: boolean | null = null;

function checkWorkerSupport(): boolean {
  if (workerSupported !== null) {
    return workerSupported;
  }

  try {
    workerSupported =
      typeof Worker !== 'undefined' && typeof import.meta.url !== 'undefined';
  } catch {
    workerSupported = false;
  }

  return workerSupported;
}

async function exportKeyForWorker(key: SessionKey): Promise<CryptoKey> {
  try {
    const keyData = await crypto.subtle.exportKey('raw', key);
    return await crypto.subtle.importKey(
      'raw',
      keyData,
      { name: 'AES-GCM' },
      true,
      ['encrypt', 'decrypt']
    );
  } catch {
    return key;
  }
}

export async function encryptFile(
  sessionKey: SessionKey,
  file: File,
  onProgress?: (progress: number) => void
): Promise<{ chunks: EncryptedChunk[]; totalSize: number }> {
  const useWorker = checkWorkerSupport() && file.size >= WORKER_THRESHOLD;

  if (useWorker) {
    try {
      const extractableKey = await exportKeyForWorker(sessionKey);
      return await encryptFileWithWorker(extractableKey, file, onProgress);
    } catch (error) {
      console.warn(
        'Worker encryption failed, falling back to main thread:',
        error
      );
    }
  }

  const arrayBuffer = await file.arrayBuffer();
  const totalSize = arrayBuffer.byteLength;
  const chunks: EncryptedChunk[] = [];
  const totalChunks = Math.ceil(totalSize / FILE_CHUNK_SIZE);

  for (let offset = 0; offset < totalSize; offset += FILE_CHUNK_SIZE) {
    const chunkEnd = Math.min(offset + FILE_CHUNK_SIZE, totalSize);
    const chunk = arrayBuffer.slice(offset, chunkEnd);
    const chunkBytes = new Uint8Array(chunk);

    const encrypted = await encryptBinary(sessionKey, chunkBytes);
    chunks.push(encrypted);

    if (onProgress && totalChunks > 1) {
      const progress = Math.round((chunks.length / totalChunks) * 100);
      onProgress(progress);
    }
  }

  return { chunks, totalSize };
}

export async function decryptFile(
  sessionKey: SessionKey,
  chunks: EncryptedChunk[],
  onProgress?: (progress: number) => void
): Promise<Blob> {
  const totalSize = chunks.reduce(
    (sum, chunk) => sum + (chunk.ciphertext?.length || 0),
    0
  );
  const useWorker = checkWorkerSupport() && totalSize >= WORKER_THRESHOLD;

  if (useWorker) {
    try {
      const extractableKey = await exportKeyForWorker(sessionKey);
      return await decryptFileWithWorker(extractableKey, chunks, onProgress);
    } catch (error) {
      console.warn(
        'Worker decryption failed, falling back to main thread:',
        error
      );
    }
  }

  const decryptedChunks: Uint8Array[] = [];
  const totalChunks = chunks.length;

  for (let i = 0; i < chunks.length; i++) {
    const chunk = chunks[i];
    const decrypted = await decryptBinary(
      sessionKey,
      chunk.ciphertext,
      chunk.nonce
    );
    decryptedChunks.push(decrypted);

    if (onProgress && totalChunks > 1) {
      const progress = Math.round(((i + 1) / totalChunks) * 100);
      onProgress(progress);
    }
  }

  const totalLength = decryptedChunks.reduce(
    (sum, chunk) => sum + chunk.length,
    0
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
  return Math.ceil(fileSize / FILE_CHUNK_SIZE);
}

export function getChunkSize(): number {
  return FILE_CHUNK_SIZE;
}
