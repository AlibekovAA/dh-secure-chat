import {
  MAX_FILE_SIZE,
  MAX_VOICE_SIZE,
  BYTES_PER_MB,
} from '@/shared/constants';

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
