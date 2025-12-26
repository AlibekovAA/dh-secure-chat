import type { EncryptedChunk } from './file-encryption';

type WorkerMessage =
  | {
      type: 'encrypt';
      fileData: ArrayBuffer;
      keyData: ArrayBuffer;
      requestId: string;
    }
  | {
      type: 'decrypt';
      chunks: EncryptedChunk[];
      keyData: ArrayBuffer;
      requestId: string;
    };

type WorkerResponse =
  | {
      type: 'encrypt-result';
      requestId: string;
      result: { chunks: EncryptedChunk[]; totalSize: number };
    }
  | { type: 'decrypt-result'; requestId: string; result: Blob }
  | { type: 'error'; requestId: string; error: string }
  | { type: 'progress'; requestId: string; progress: number };

let worker: Worker | null = null;
let requestCounter = 0;
const pendingRequests = new Map<
  string,
  {
    resolve: (value: any) => void;
    reject: (error: Error) => void;
    onProgress?: (progress: number) => void;
  }
>();

function getWorker(): Worker {
  if (!worker) {
    worker = new Worker(
      new URL('./file-encryption.worker.ts', import.meta.url),
      { type: 'module' },
    );

    worker.onmessage = (e: MessageEvent<WorkerResponse>) => {
      const { requestId, type } = e.data;

      const pending = pendingRequests.get(requestId);
      if (!pending) return;

      if (type === 'encrypt-result') {
        pending.resolve(e.data.result);
        pendingRequests.delete(requestId);
      } else if (type === 'decrypt-result') {
        pending.resolve(e.data.result);
        pendingRequests.delete(requestId);
      } else if (type === 'error') {
        pending.reject(new Error(e.data.error));
        pendingRequests.delete(requestId);
      } else if (type === 'progress') {
        pending.onProgress?.(e.data.progress);
      }
    };

    worker.onerror = (error) => {
      console.error('File encryption worker error:', error);
      for (const [requestId, pending] of pendingRequests.entries()) {
        pending.reject(new Error('Worker error'));
        pendingRequests.delete(requestId);
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
  onProgress?: (progress: number) => void,
): Promise<{ chunks: EncryptedChunk[]; totalSize: number }> {
  const requestId = `encrypt-${++requestCounter}`;
  const fileData = await file.arrayBuffer();
  const keyData = await exportKey(sessionKey);

  return new Promise((resolve, reject) => {
    pendingRequests.set(requestId, { resolve, reject, onProgress });

    const worker = getWorker();
    worker.postMessage(
      {
        type: 'encrypt',
        fileData,
        keyData,
        requestId,
      } as WorkerMessage,
      [fileData, keyData],
    );
  });
}

export async function decryptFileWithWorker(
  sessionKey: CryptoKey,
  chunks: EncryptedChunk[],
  onProgress?: (progress: number) => void,
): Promise<Blob> {
  const requestId = `decrypt-${++requestCounter}`;
  const keyData = await exportKey(sessionKey);

  return new Promise((resolve, reject) => {
    pendingRequests.set(requestId, { resolve, reject, onProgress });

    const worker = getWorker();
    worker.postMessage(
      {
        type: 'decrypt',
        chunks,
        keyData,
        requestId,
      } as WorkerMessage,
      [keyData],
    );
  });
}

export function terminateWorker(): void {
  if (worker) {
    worker.terminate();
    worker = null;
    for (const [requestId, pending] of pendingRequests.entries()) {
      pending.reject(new Error('Worker terminated'));
      pendingRequests.delete(requestId);
    }
  }
}
