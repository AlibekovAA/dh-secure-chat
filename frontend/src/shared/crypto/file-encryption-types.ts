import type { EncryptedChunk } from './file-encryption';

export type WorkerMessage =
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

export type WorkerResponse =
  | {
      type: 'encrypt-result';
      requestId: string;
      result: { chunks: EncryptedChunk[]; totalSize: number };
    }
  | { type: 'decrypt-result'; requestId: string; result: Blob }
  | { type: 'error'; requestId: string; error: string }
  | { type: 'progress'; requestId: string; progress: number };
