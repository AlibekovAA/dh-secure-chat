import { FILE_CHUNK_SIZE } from '../constants';
import {
  encryptBinaryWithKey,
  decryptBinaryWithKey,
} from './binary-encryption';
import type { EncryptedChunk } from './file-encryption';
import type { WorkerMessage, WorkerResponse } from './file-encryption-types';

async function importKey(keyData: ArrayBuffer): Promise<CryptoKey> {
  return await crypto.subtle.importKey(
    'raw',
    keyData,
    { name: 'AES-GCM' },
    false,
    ['encrypt', 'decrypt'],
  );
}

async function handleEncrypt(
  fileData: ArrayBuffer,
  keyData: ArrayBuffer,
  requestId: string,
): Promise<void> {
  try {
    const sessionKey = await importKey(keyData);
    const totalSize = fileData.byteLength;
    const chunks: EncryptedChunk[] = [];
    const totalChunks = Math.ceil(totalSize / FILE_CHUNK_SIZE);

    for (let offset = 0; offset < totalSize; offset += FILE_CHUNK_SIZE) {
      const chunkEnd = Math.min(offset + FILE_CHUNK_SIZE, totalSize);
      const chunk = fileData.slice(offset, chunkEnd);
      const chunkBytes = new Uint8Array(chunk);

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
  requestId: string,
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
        chunk.nonce,
      );
      decryptedChunks.push(decrypted);

      const progress = Math.round(((i + 1) / totalChunks) * 100);
      self.postMessage({
        type: 'progress',
        requestId,
        progress,
      } as WorkerResponse);
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

    const blob = new Blob([result]);

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
    await handleEncrypt(e.data.fileData, e.data.keyData, requestId);
  } else if (type === 'decrypt') {
    await handleDecrypt(e.data.chunks, e.data.keyData, requestId);
  }
};
