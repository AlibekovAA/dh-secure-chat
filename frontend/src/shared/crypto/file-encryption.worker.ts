import {
  encryptBinaryWithKey,
  decryptBinaryWithKey,
} from '@/shared/crypto/binary-encryption';
import {
  decryptFileChunks,
  encryptFileChunks,
} from '@/shared/crypto/file-chunks';
import type {
  EncryptedChunk,
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
    const result = await encryptFileChunks(
      sessionKey,
      file,
      encryptBinaryWithKey,
      (progress) => {
        self.postMessage({
          type: 'progress',
          requestId,
          progress,
        } as WorkerResponse);
      },
      true
    );

    self.postMessage({
      type: 'encrypt-result',
      requestId,
      result,
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
    const result = await decryptFileChunks(
      sessionKey,
      chunks,
      decryptBinaryWithKey,
      (progress) => {
        self.postMessage({
          type: 'progress',
          requestId,
          progress,
        } as WorkerResponse);
      },
      true
    );

    self.postMessage({
      type: 'decrypt-result',
      requestId,
      result,
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
