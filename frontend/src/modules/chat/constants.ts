export const MAX_FILE_SIZE = 50 * 1024 * 1024;

export const MAX_VOICE_DURATION_SECONDS = 5 * 60;
export const MAX_VOICE_SIZE = 10 * 1024 * 1024;
export const VOICE_MIME_TYPES = [
  'audio/webm',
  'audio/ogg',
  'audio/mpeg',
  'audio/mp4',
  'audio/webm;codecs=opus',
  'audio/ogg;codecs=opus',
];

export const VIDEO_MIME_TYPES = [
  'video/mp4',
  'video/webm',
  'video/ogg',
  'video/quicktime',
  'video/x-msvideo',
  'video/x-matroska',
];

export const IMAGE_MIME_TYPES = [
  'image/jpeg',
  'image/png',
  'image/webp',
  'image/gif',
  'image/bmp',
  'image/svg+xml',
];

export const PDF_MIME_TYPE = 'application/pdf';

export const TEXT_MIME_TYPES = [
  'text/plain',
  'text/html',
  'text/css',
  'text/javascript',
  'text/json',
  'application/json',
];

export const MAX_MESSAGE_LENGTH = 4000;
export const VIDEO_THUMBNAIL_SIZE = 200;
export const VIDEO_THUMBNAIL_CACHE_PREFIX = 'video_thumbnail_';
export const VIDEO_THUMBNAIL_CACHE_DURATION_MS = 1 * 24 * 60 * 60 * 1000;
