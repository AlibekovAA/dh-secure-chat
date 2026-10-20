import { FILE_CHUNK_SIZE } from '@/shared/constants';
import {
  encryptBinaryWithKey,
  decryptBinaryWithKey,
} from '@/shared/crypto/binary-encryption';
import type { EncryptedChunk } from '@/shared/crypto/file-encryption';
import type {
  WorkerMessage,
  WorkerResponse,
} from '@/shared/crypto/file-encryption-types';

async function importKey(keyData: ArrayBuffer): Promise<CryptoKey> {
  return await crypto.subtle.importKey(
    'raw',
    keyData,
    { name: 'AES-GCM' },
    false,
    ['encrypt', 'decrypt']
  );
}

async function handleEncrypt(
  file: Blob,
  keyData: ArrayBuffer,
  requestId: string
): Promise<void> {
  try {
    const sessionKey = await importKey(keyData);
    const totalSize = file.size;
    const chunks: EncryptedChunk[] = [];
    const totalChunks = Math.ceil(totalSize / FILE_CHUNK_SIZE);

    for (let offset = 0; offset < totalSize; offset += FILE_CHUNK_SIZE) {
      const chunkEnd = Math.min(offset + FILE_CHUNK_SIZE, totalSize);
      const chunkBytes = new Uint8Array(
        await file.slice(offset, chunkEnd).arrayBuffer()
      );

      const encrypted = await encryptBinaryWithKey(sessionKey, chunkBytes);
      chunks.push(encrypted);

      const progress = Math.round((chunks.length / totalChunks) * 100);
      self.postMessage({
        type: 'progress',
        requestId,
        progress,
      } as WorkerResponse);
    }

    self.postMessage({
      type: 'encrypt-result',
      requestId,
      result: { chunks, totalSize },
    } as WorkerResponse);
  } catch (error) {
    self.postMessage({
      type: 'error',
      requestId,
      error: error instanceof Error ? error.message : 'Encryption failed',
    } as WorkerResponse);
  }
}

async function handleDecrypt(
  chunks: EncryptedChunk[],
  keyData: ArrayBuffer,
  requestId: string
): Promise<void> {
  try {
    const sessionKey = await importKey(keyData);
    const decryptedChunks: Uint8Array[] = [];
    const totalChunks = chunks.length;

    for (let i = 0; i < chunks.length; i++) {
      const chunk = chunks[i];
      const decrypted = await decryptBinaryWithKey(
        sessionKey,
        chunk.ciphertext,
        chunk.nonce
      );
      decryptedChunks.push(decrypted);

      const progress = Math.round(((i + 1) / totalChunks) * 100);
      self.postMessage({
        type: 'progress',
        requestId,
        progress,
      } as WorkerResponse);
    }

    const blob = new Blob(decryptedChunks as BlobPart[]);

    self.postMessage({
      type: 'decrypt-result',
      requestId,
      result: blob,
    } as WorkerResponse);
  } catch (error) {
    self.postMessage({
      type: 'error',
      requestId,
      error: error instanceof Error ? error.message : 'Decryption failed',
    } as WorkerResponse);
  }
}

self.onmessage = async (e: MessageEvent<WorkerMessage>) => {
  const { type, requestId } = e.data;

  if (type === 'encrypt') {
    await handleEncrypt(e.data.file, e.data.keyData, requestId);
  } else if (type === 'decrypt') {
    await handleDecrypt(e.data.chunks, e.data.keyData, requestId);
  }
};
