import {
  encryptBinary,
  decryptBinary,
} from '@/shared/crypto/binary-encryption';
import type { SessionKey } from '@/shared/crypto/session';
import {
  encryptFileWithWorker,
  decryptFileWithWorker,
} from '@/shared/crypto/file-encryption-worker';
import {
  decryptFileChunks,
  encryptFileChunks,
} from '@/shared/crypto/file-chunks';
import { FILE_CHUNK_SIZE, WORKER_THRESHOLD } from '@/shared/constants';
import type { EncryptedChunk } from '@/shared/crypto/file-encryption-types';

export type { EncryptedChunk } from '@/shared/crypto/file-encryption-types';

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
    } catch (_error) {
      void _error;
    }
  }

  return encryptFileChunks(
    sessionKey,
    file,
    (key, chunk) => encryptBinary(key as SessionKey, chunk),
    onProgress
  );
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
    } catch (_error) {
      void _error;
    }
  }

  return decryptFileChunks(
    sessionKey,
    chunks,
    (key, ciphertext, nonce) =>
      decryptBinary(key as SessionKey, ciphertext, nonce),
    onProgress
  );
}

export function calculateChunks(fileSize: number): number {
  return Math.ceil(fileSize / FILE_CHUNK_SIZE);
}

export function getChunkSize(): number {
  return FILE_CHUNK_SIZE;
}
