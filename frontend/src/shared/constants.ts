export const FILE_CHUNK_SIZE = 1024 * 1024;
export const BASE64_CHUNK_SIZE = 8192;
export const AES_KEY_SIZE = 256;
export const WORKER_THRESHOLD = 5 * 1024 * 1024;

export const MAX_RECONNECT_DELAY_MS = 30000;
export const MAX_SEQUENCES_TO_KEEP = 10000;

export const HTTP_SERVER_ERROR_THRESHOLD = 500;

export const MAX_FILE_SIZE = 50 * 1024 * 1024;

export const MAX_VOICE_DURATION_SECONDS = 5 * 60;
export const MAX_VOICE_SIZE = 10 * 1024 * 1024;

export const VIDEO_THUMBNAIL_REQUEST_IDLE_TIMEOUT_MS = 500;
export const VIDEO_THUMBNAIL_INTERSECTION_THRESHOLD = 0.1;
export const VIDEO_THUMBNAIL_ROOT_MARGIN = '50px';

export const EMOJI_PICKER_WIDTH = 300;
export const EMOJI_PICKER_ESTIMATED_WIDTH = EMOJI_PICKER_WIDTH;
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
export const EDIT_TIMEOUT_MS = 15 * 60 * 1000;

export const EMOJI_LIST = [
  '👍',
  '❤️',
  '😂',
  '😮',
  '😢',
  '🙏',
  '🔥',
  '👏',
  '🎉',
  '💯',
  '😍',
  '🤔',
  '👎',
  '😡',
  '🤝',
  '💪',
];

export const RESULTS_PER_PAGE = 4;

export const MENU_WIDTH = 260;
export const MENU_ESTIMATED_WIDTH = MENU_WIDTH;
export const MENU_ESTIMATED_HEIGHT = 140;
export const MENU_PADDING = 10;

export const ACK_TIMEOUT_MS = 5000;
export const ACK_MAX_RETRIES = 3;
export const ACK_RETRY_DELAY_MS = 1000;

export const DB_NAME = 'secure-chat-db';
export const DB_VERSION = 1;
export const DB_STORE_NAME = 'keys';

export const IDENTITY_KEY_STORAGE = 'identity_private_key';
export const MASTER_KEY_STORAGE = 'identity_master_key';

export const VERIFIED_PEERS_STORAGE = 'verified_peers';
export const FINGERPRINT_HISTORY_STORAGE = 'fingerprint_history';

export const TOKEN_STORAGE_KEY = 'auth_token';

export const WEBSOCKET_MAX_RECONNECT_ATTEMPTS = 5;
export const WEBSOCKET_BASE_DELAY_MS = 1000;

export const UNAUTHORIZED_MESSAGE = 'unauthorized';
export const SESSION_EXPIRED_ERROR = 'session_expired';

export const MS_PER_SECOND = 1000;
export const BYTES_PER_MB = 1024 * 1024;

export const TYPING_INDICATOR_TIMEOUT_MS = 3000;

export const VIDEO_THUMBNAIL_SEEK_TIME = 0.1;
export const JPEG_QUALITY = 0.85;
export const VIDEO_THUMBNAIL_BG_COLOR = '#0a0a0a';

export const INPUT_MIN_HEIGHT_PX = 40;
export const MODAL_MAX_HEIGHT_VH = 80;
export const MESSAGE_MAX_WIDTH_PERCENT = 75;
export const TEXTAREA_MAX_ROWS = 5;

export const USERNAME_MIN_LENGTH = 3;
export const USERNAME_MAX_LENGTH = 32;
export const PASSWORD_MIN_LENGTH = 8;
export const PASSWORD_MAX_LENGTH = 72;
export const USERNAME_REGEX = /^[a-zA-Z0-9_-]+$/;

export const MAX_SEARCH_QUERY_LENGTH = 100;
