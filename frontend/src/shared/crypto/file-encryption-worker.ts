import type {
  WorkerMessage,
  WorkerResponse,
} from '@/shared/crypto/file-encryption-types';
import type { EncryptedChunk } from '@/shared/crypto/file-encryption';

let worker: Worker | null = null;
let requestCounter = 0;
const pendingEncryptRequests = new Map<
  string,
  {
    resolve: (value: { chunks: EncryptedChunk[]; totalSize: number }) => void;
    reject: (error: Error) => void;
    onProgress?: (progress: number) => void;
  }
>();
const pendingDecryptRequests = new Map<
  string,
  {
    resolve: (value: Blob) => void;
    reject: (error: Error) => void;
    onProgress?: (progress: number) => void;
  }
>();

function getWorker(): Worker {
  if (!worker) {
    worker = new Worker(
      new URL('./file-encryption.worker.ts', import.meta.url),
      { type: 'module' }
    );

    worker.onmessage = (e: MessageEvent<WorkerResponse>) => {
      const { requestId, type } = e.data;

      if (type === 'encrypt-result') {
        const pending = pendingEncryptRequests.get(requestId);
        if (pending) {
          pending.resolve(e.data.result);
          pendingEncryptRequests.delete(requestId);
        }
      } else if (type === 'decrypt-result') {
        const pending = pendingDecryptRequests.get(requestId);
        if (pending) {
          pending.resolve(e.data.result);
          pendingDecryptRequests.delete(requestId);
        }
      } else if (type === 'error') {
        const pendingEncrypt = pendingEncryptRequests.get(requestId);
        const pendingDecrypt = pendingDecryptRequests.get(requestId);
        const pending = pendingEncrypt || pendingDecrypt;
        if (pending) {
          pending.reject(new Error(e.data.error));
          pendingEncryptRequests.delete(requestId);
          pendingDecryptRequests.delete(requestId);
        }
      } else if (type === 'progress') {
        const pendingEncrypt = pendingEncryptRequests.get(requestId);
        const pendingDecrypt = pendingDecryptRequests.get(requestId);
        pendingEncrypt?.onProgress?.(e.data.progress);
        pendingDecrypt?.onProgress?.(e.data.progress);
      }
    };

    worker.onerror = (error) => {
      console.error('File encryption worker error:', error);
      for (const [requestId, pending] of pendingEncryptRequests.entries()) {
        pending.reject(new Error('Worker error'));
        pendingEncryptRequests.delete(requestId);
      }
      for (const [requestId, pending] of pendingDecryptRequests.entries()) {
        pending.reject(new Error('Worker error'));
        pendingDecryptRequests.delete(requestId);
      }
    };
  }

  return worker;
}

async function exportKey(key: CryptoKey): Promise<ArrayBuffer> {
  return await crypto.subtle.exportKey('raw', key);
}

export async function encryptFileWithWorker(
  sessionKey: CryptoKey,
  file: File,
  onProgress?: (progress: number) => void
): Promise<{ chunks: EncryptedChunk[]; totalSize: number }> {
  const requestId = `encrypt-${++requestCounter}`;
  const fileData = await file.arrayBuffer();
  const keyData = await exportKey(sessionKey);

  return new Promise((resolve, reject) => {
    pendingEncryptRequests.set(requestId, { resolve, reject, onProgress });

    const worker = getWorker();
    worker.postMessage(
      {
        type: 'encrypt',
        fileData,
        keyData,
        requestId,
      } as WorkerMessage,
      [fileData, keyData]
    );
  });
}

export async function decryptFileWithWorker(
  sessionKey: CryptoKey,
  chunks: EncryptedChunk[],
  onProgress?: (progress: number) => void
): Promise<Blob> {
  const requestId = `decrypt-${++requestCounter}`;
  const keyData = await exportKey(sessionKey);

  return new Promise((resolve, reject) => {
    pendingDecryptRequests.set(requestId, { resolve, reject, onProgress });

    const worker = getWorker();
    worker.postMessage(
      {
        type: 'decrypt',
        chunks,
        keyData,
        requestId,
      } as WorkerMessage,
      [keyData]
    );
  });
}
