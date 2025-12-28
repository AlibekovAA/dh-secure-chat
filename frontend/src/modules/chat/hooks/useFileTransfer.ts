import { useCallback } from 'react';
import type { FileStartPayload } from '../../../shared/websocket/types';
import { decryptFile } from '../../../shared/crypto/file-encryption';
import type { SessionKey } from '../../../shared/crypto/session';
import type { ChatMessage } from '../useChatSession';
import { extractDurationFromFilename } from '../utils';
import type { WSMessage } from '../../../shared/websocket/types';

type FileBuffer = {
  chunks: Array<{ ciphertext: string; nonce: string }>;
  metadata: FileStartPayload;
  processing: boolean;
};

type UseFileTransferOptions = {
  sessionKeyRef: React.MutableRefObject<SessionKey | null>;
  fileBuffersRef: React.MutableRefObject<Map<string, FileBuffer>>;
  setMessages: React.Dispatch<React.SetStateAction<ChatMessage[]>>;
  setError: (error: string | null) => void;
  sendRef: React.MutableRefObject<((message: WSMessage) => void) | null>;
  peerId: string | null;
};

export function useFileTransfer({
  sessionKeyRef,
  fileBuffersRef,
  setMessages,
  setError,
  sendRef,
  peerId,
}: UseFileTransferOptions) {
  const handleIncomingFile = useCallback(
    async (fileId: string) => {
      if (!sessionKeyRef.current) {
        return;
      }

      const buffer = fileBuffersRef.current.get(fileId);
      if (!buffer) {
        return;
      }

      if (buffer.processing) {
        return;
      }

      const { chunks, metadata } = buffer;
      const expectedChunks = metadata.total_chunks;

      if (chunks.length < expectedChunks) {
        return;
      }

      buffer.processing = true;

      const sortedChunks: Array<{ ciphertext: string; nonce: string }> = [];
      for (let i = 0; i < expectedChunks; i++) {
        const chunk = chunks[i];
        if (!chunk || !chunk.ciphertext || !chunk.nonce) {
          setError(
            `Не все части файла получены (${sortedChunks.length}/${expectedChunks})`,
          );
          return;
        }
        sortedChunks.push(chunk);
      }

      try {
        const decryptedBlob = await decryptFile(
          sessionKeyRef.current,
          sortedChunks,
        );

        if (!decryptedBlob || decryptedBlob.size === 0) {
          setError('Получен пустой файл');
          fileBuffersRef.current.delete(fileId);
          return;
        }

        const mimeType = metadata.mime_type || 'application/octet-stream';
        const blob = new Blob([decryptedBlob], { type: mimeType });

        if (!blob || blob.size === 0) {
          setError('Получен пустой файл');
          fileBuffersRef.current.delete(fileId);
          return;
        }

        const isVoice =
          metadata.mime_type && metadata.mime_type.startsWith('audio/');
        const extractedDuration = extractDurationFromFilename(
          metadata.filename,
        );

        const newMessage: ChatMessage = {
          id: fileId,
          ...(isVoice
            ? {
                voice: {
                  filename: metadata.filename,
                  mimeType,
                  size: metadata.total_size,
                  duration: extractedDuration > 0 ? extractedDuration : 0,
                  blob,
                },
              }
            : {
                file: {
                  filename: metadata.filename,
                  mimeType,
                  size: metadata.total_size,
                  blob,
                  accessMode: metadata.access_mode || 'both',
                },
              }),
          timestamp: Date.now(),
          isOwn: false,
        };

        setMessages((prev) => [...prev, newMessage]);
        fileBuffersRef.current.delete(fileId);

        if (sendRef.current && peerId && metadata.from) {
          sendRef.current({
            type: 'ack',
            payload: {
              to: metadata.from,
              message_id: fileId,
            },
          });
        }
      } catch (err) {
        setError(
          'Не удалось расшифровать файл. Возможно, сессия была прервана.',
        );
        fileBuffersRef.current.delete(fileId);
      }
    },
    [sessionKeyRef, setMessages, setError, sendRef, peerId],
  );

  const clearFileBuffers = useCallback(() => {
    fileBuffersRef.current.clear();
  }, []);

  return {
    handleIncomingFile,
    clearFileBuffers,
  };
}
