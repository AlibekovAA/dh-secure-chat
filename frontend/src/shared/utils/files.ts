import {
  IMAGE_MIME_TYPES,
  MAX_FILE_SIZE,
  MAX_VOICE_SIZE,
  VIDEO_MIME_TYPES,
  BYTES_PER_MB,
} from '@/shared/constants';

export type FileContentKind =
  'image' | 'video' | 'audio' | 'document' | 'other';

export type FileContentInspection = {
  detectedMimeType: string | null;
  kind: FileContentKind;
  previewAllowed: boolean;
};

const HEADER_SIZE = 512;

function startsWith(bytes: Uint8Array, signature: number[]): boolean {
  return signature.every((value, index) => bytes[index] === value);
}

function readAscii(bytes: Uint8Array, start: number, length: number): string {
  return String.fromCharCode(...bytes.slice(start, start + length));
}

function detectImageMimeType(bytes: Uint8Array): string | null {
  if (startsWith(bytes, [0xff, 0xd8, 0xff])) return 'image/jpeg';
  if (startsWith(bytes, [0x89, 0x50, 0x4e, 0x47, 0x0d, 0x0a, 0x1a, 0x0a]))
    return 'image/png';
  if (
    readAscii(bytes, 0, 6) === 'GIF87a' ||
    readAscii(bytes, 0, 6) === 'GIF89a'
  )
    return 'image/gif';
  if (readAscii(bytes, 0, 4) === 'RIFF' && readAscii(bytes, 8, 4) === 'WEBP')
    return 'image/webp';
  if (readAscii(bytes, 0, 2) === 'BM') return 'image/bmp';

  const text = new TextDecoder()
    .decode(bytes)
    .replace(/^\uFEFF/, '')
    .trimStart();
  if (/^(?:<\?xml[^>]*>\s*)?<svg(?:\s|>)/i.test(text)) return 'image/svg+xml';
  return null;
}

function detectMediaMimeType(
  bytes: Uint8Array,
  declaredMimeType: string
): { mimeType: string; kind: 'audio' | 'video' } | null {
  const declaredKind = declaredMimeType.startsWith('audio/')
    ? 'audio'
    : 'video';

  if (readAscii(bytes, 0, 4) === 'OggS') {
    return {
      mimeType: declaredKind === 'audio' ? declaredMimeType : 'video/ogg',
      kind: declaredKind,
    };
  }
  if (startsWith(bytes, [0x1a, 0x45, 0xdf, 0xa3])) {
    return {
      mimeType: declaredKind === 'audio' ? declaredMimeType : 'video/webm',
      kind: declaredKind,
    };
  }
  if (readAscii(bytes, 4, 4) === 'ftyp') {
    return {
      mimeType: declaredKind === 'audio' ? declaredMimeType : 'video/mp4',
      kind: declaredKind,
    };
  }
  if (readAscii(bytes, 0, 4) === 'RIFF' && readAscii(bytes, 8, 4) === 'AVI ')
    return { mimeType: 'video/x-msvideo', kind: 'video' };
  if (readAscii(bytes, 0, 4) === 'RIFF' && readAscii(bytes, 8, 4) === 'WAVE')
    return { mimeType: declaredMimeType || 'audio/wav', kind: 'audio' };
  if (
    readAscii(bytes, 0, 3) === 'ID3' ||
    (bytes[0] === 0xff && (bytes[1] & 0xe0) === 0xe0)
  )
    return { mimeType: declaredMimeType || 'audio/mpeg', kind: 'audio' };

  return null;
}

function isPlainText(bytes: Uint8Array): boolean {
  if (bytes.includes(0)) return false;

  try {
    new TextDecoder('utf-8', { fatal: true }).decode(bytes);
    return true;
  } catch {
    return false;
  }
}

export async function inspectFileContent(
  blob: Blob,
  declaredMimeType: string
): Promise<FileContentInspection> {
  if (blob.size === 0) {
    return { detectedMimeType: null, kind: 'other', previewAllowed: false };
  }

  const bytes = new Uint8Array(
    await blob.slice(0, Math.min(blob.size, HEADER_SIZE)).arrayBuffer()
  );
  if (readAscii(bytes, 0, 5) === '%PDF-') {
    return {
      detectedMimeType: 'application/pdf',
      kind: 'document',
      previewAllowed: true,
    };
  }

  if (declaredMimeType === 'text/plain' && isPlainText(bytes)) {
    return {
      detectedMimeType: 'text/plain',
      kind: 'document',
      previewAllowed: true,
    };
  }

  const imageMimeType = detectImageMimeType(bytes);
  if (imageMimeType && IMAGE_MIME_TYPES.includes(imageMimeType)) {
    return {
      detectedMimeType: imageMimeType,
      kind: 'image',
      previewAllowed: true,
    };
  }

  const media = detectMediaMimeType(bytes, declaredMimeType);
  if (media) {
    const supported =
      media.kind === 'audio' || VIDEO_MIME_TYPES.includes(media.mimeType);
    return {
      detectedMimeType: supported ? media.mimeType : null,
      kind: supported ? media.kind : 'other',
      previewAllowed: supported,
    };
  }

  return { detectedMimeType: null, kind: 'other', previewAllowed: false };
}

export type FileValidationError =
  | { type: 'empty'; message: string }
  | {
      type: 'too_large';
      message: string;
      fileSizeMB: number;
      maxSizeMB: number;
    }
  | {
      type: 'voice_too_large';
      message: string;
      fileSizeMB: number;
      maxSizeMB: number;
    };

export type FileValidationOptions = {
  maxSize?: number;
  emptyAllowed?: boolean;
  fileType?: 'file' | 'image' | 'voice';
};

export function validateFileSize(
  file: File,
  options: FileValidationOptions = {}
): FileValidationError | null {
  const {
    maxSize = MAX_FILE_SIZE,
    emptyAllowed = false,
    fileType = 'file',
  } = options;

  if (!emptyAllowed && file.size === 0) {
    const messages = {
      file: 'Файл пустой. Выберите файл с содержимым.',
      image: 'Файл пустой. Выберите файл с содержимым.',
      voice: 'Голосовое сообщение пустое. Запишите сообщение с звуком.',
    };
    return {
      type: 'empty',
      message: messages[fileType],
    };
  }

  if (file.size > maxSize) {
    const fileSizeMB = file.size / BYTES_PER_MB;
    const maxSizeMB = maxSize / BYTES_PER_MB;
    const messages = {
      file: `Файл слишком большой. Максимальный размер: ${maxSizeMB.toFixed(0)}MB. Выберите файл меньшего размера.`,
      image: `Изображение слишком большое. Максимальный размер: ${maxSizeMB.toFixed(0)}MB. Выберите изображение меньшего размера.`,
      voice: `Голосовое сообщение слишком большое. Максимальный размер: ${maxSizeMB.toFixed(0)}MB. Запишите более короткое сообщение.`,
    };
    return {
      type: 'too_large',
      message: messages[fileType],
      fileSizeMB,
      maxSizeMB,
    };
  }

  return null;
}

export function validateVoiceSize(file: File): FileValidationError | null {
  return validateFileSize(file, {
    maxSize: MAX_VOICE_SIZE,
    emptyAllowed: false,
    fileType: 'voice',
  });
}

export function validateImagePaste(file: File): FileValidationError | null {
  return validateFileSize(file, {
    maxSize: MAX_FILE_SIZE,
    emptyAllowed: false,
    fileType: 'image',
  });
}

export function getFileValidationError(
  error: FileValidationError | null
): string | null {
  if (!error) return null;
  return error.message;
}
