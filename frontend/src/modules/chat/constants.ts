export const MAX_FILE_SIZE = 50 * 1024 * 1024;

export const MAX_VOICE_DURATION_SECONDS = 5 * 60;
export const MAX_VOICE_SIZE = 10 * 1024 * 1024;

export const VIDEO_THUMBNAIL_REQUEST_IDLE_TIMEOUT_MS = 500;
export const VIDEO_THUMBNAIL_INTERSECTION_THRESHOLD = 0.1;
export const VIDEO_THUMBNAIL_ROOT_MARGIN = '50px';

export const EMOJI_PICKER_ESTIMATED_WIDTH = 220;
export const EMOJI_PICKER_ESTIMATED_HEIGHT = 120;
export const EMOJI_PICKER_PAGE_SIZE = 4;
export const EMOJI_PICKER_PADDING = 10;

export const VIDEO_RECORDER_CHECK_INTERVAL_MS = 500;
export const VIDEO_RECORDER_TIMESLICE_MS = 1000;
export const VIDEO_RECORDER_DURATION_UPDATE_INTERVAL_MS = 100;
export const VIDEO_RECORDER_DURATION_UPDATE_DELAY_MS = 100;

export const MESSAGE_READ_INTERSECTION_THRESHOLD = 0.5;
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
