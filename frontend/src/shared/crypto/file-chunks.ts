import { FILE_CHUNK_SIZE } from '@/shared/constants';
import type { EncryptedChunk } from '@/shared/crypto/file-encryption-types';

type EncryptChunk = (
  key: CryptoKey,
  data: Uint8Array
) => Promise<EncryptedChunk>;

type DecryptChunk = (
  key: CryptoKey,
  ciphertext: string,
  nonce: string
) => Promise<Uint8Array>;

function reportProgress(
  onProgress: ((progress: number) => void) | undefined,
  completedChunks: number,
  totalChunks: number,
  reportSingleChunk: boolean
): void {
  if (onProgress && (reportSingleChunk || totalChunks > 1)) {
    onProgress(Math.round((completedChunks / totalChunks) * 100));
  }
}

export async function encryptFileChunks(
  key: CryptoKey,
  file: Blob,
  encryptChunk: EncryptChunk,
  onProgress?: (progress: number) => void,
  reportSingleChunk = false
): Promise<{ chunks: EncryptedChunk[]; totalSize: number }> {
  const totalSize = file.size;
  const chunks: EncryptedChunk[] = [];
  const totalChunks = Math.ceil(totalSize / FILE_CHUNK_SIZE);

  for (let offset = 0; offset < totalSize; offset += FILE_CHUNK_SIZE) {
    const chunkEnd = Math.min(offset + FILE_CHUNK_SIZE, totalSize);
    const chunkBytes = new Uint8Array(
      await file.slice(offset, chunkEnd).arrayBuffer()
    );
    chunks.push(await encryptChunk(key, chunkBytes));
    reportProgress(onProgress, chunks.length, totalChunks, reportSingleChunk);
  }

  return { chunks, totalSize };
}

export async function decryptFileChunks(
  key: CryptoKey,
  chunks: EncryptedChunk[],
  decryptChunk: DecryptChunk,
  onProgress?: (progress: number) => void,
  reportSingleChunk = false
): Promise<Blob> {
  const decryptedChunks: Uint8Array[] = [];
  const totalChunks = chunks.length;

  for (let index = 0; index < totalChunks; index++) {
    const chunk = chunks[index];
    decryptedChunks.push(
      await decryptChunk(key, chunk.ciphertext, chunk.nonce)
    );
    reportProgress(onProgress, index + 1, totalChunks, reportSingleChunk);
  }

  return new Blob(decryptedChunks as BlobPart[]);
}
