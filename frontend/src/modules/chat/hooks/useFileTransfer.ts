import { useCallback } from 'react';
import { decryptFile } from '@/shared/crypto/file-encryption';
import type { SessionKey } from '@/shared/crypto/session';
import type { ChatMessage, FileBuffer } from '@/modules/chat/useChatSession';
import { extractDurationFromFilename } from '@/modules/chat/utils';
import type { WSMessage } from '@/shared/websocket/types';
import { MESSAGES } from '@/shared/messages';
import { inspectFileContent } from '@/shared/utils/files';

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

      if (
        chunks.length < expectedChunks ||
        buffer.receivedSize !== metadata.total_size
      ) {
        return;
      }

      buffer.processing = true;

      const sortedChunks: Array<{ ciphertext: string; nonce: string }> = [];
      for (let i = 0; i < expectedChunks; i++) {
        const chunk = chunks[i];
        if (!chunk || !chunk.ciphertext || !chunk.nonce) {
          setError(
            MESSAGES.chat.fileTransfer.errors.missingChunks(
              sortedChunks.length,
              expectedChunks
            )
          );
          return;
        }
        sortedChunks.push(chunk);
      }

      try {
        const decryptedBlob = await decryptFile(
          sessionKeyRef.current,
          sortedChunks
        );

        if (!decryptedBlob || decryptedBlob.size !== metadata.total_size) {
          setError(MESSAGES.chat.fileTransfer.errors.invalidTransfer);
          fileBuffersRef.current.delete(fileId);
          return;
        }

        const inspection = await inspectFileContent(
          decryptedBlob,
          metadata.mime_type
        );
        const mimeType =
          inspection.previewAllowed && inspection.detectedMimeType
            ? inspection.detectedMimeType
            : 'application/octet-stream';
        const blob = new Blob([decryptedBlob], { type: mimeType });

        if (!blob || blob.size === 0) {
          setError(MESSAGES.chat.fileTransfer.errors.emptyFile);
          fileBuffersRef.current.delete(fileId);
          return;
        }

        const canPreview =
          inspection.previewAllowed && metadata.access_mode !== 'download_only';
        const isVoice = canPreview && inspection.kind === 'audio';
        const isVideo = canPreview && inspection.kind === 'video';
        const extractedDuration = extractDurationFromFilename(
          metadata.filename
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
            : isVideo
              ? {
                  video: {
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
                    previewAllowed: inspection.previewAllowed,
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
        setError(MESSAGES.chat.fileTransfer.errors.decryptFailed);
        fileBuffersRef.current.delete(fileId);
      }
    },
    [sessionKeyRef, setMessages, setError, sendRef, peerId, fileBuffersRef]
  );

  const clearFileBuffers = useCallback(() => {
    fileBuffersRef.current.clear();
  }, [fileBuffersRef]);

  return {
    handleIncomingFile,
    clearFileBuffers,
  };
}
